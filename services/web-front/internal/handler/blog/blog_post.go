package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"
	"seungpyo.lee/PersonalWebSite/services/web-front/internal/config"
)

type BlogPostHandler interface {
	Save(c *gin.Context)
}

type postHandler struct {
	cfg *config.PostConfig
}

func NewPostHandler(cfg *config.PostConfig) BlogPostHandler {
	return &postHandler{cfg: cfg}
}

func (h *postHandler) Save(c *gin.Context) {
	apiGatewayURL := h.cfg.ApiGatewayURL
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	title := c.PostForm("article-title")
	published := true

	content := c.PostForm("article-content")
	removeContentString := []string{"<select class=\"ql-ui\" contenteditable=\"false\"><option value=\"plain\">Plain</option><option value=\"bash\">Bash</option><option value=\"cpp\">C++</option><option value=\"cs\">C#</option><option value=\"css\">CSS</option><option value=\"diff\">Diff</option><option value=\"xml\">HTML/XML</option><option value=\"java\">Java</option><option value=\"javascript\">JavaScript</option><option value=\"markdown\">Markdown</option><option value=\"php\">PHP</option><option value=\"python\">Python</option><option value=\"ruby\">Ruby</option><option value=\"sql\">SQL</option></select>", "contenteditable=\"true\""}
	for _, value := range removeContentString {
		content = strings.Replace(content, value, "", -1)
	}

	accessToken, err := c.Cookie("access_token")
	if err != nil || accessToken == "" {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Need to Login"))
		return
	}
	// Prefer X-User-Id header (set by gateway/middleware); fallback to cookie set at login
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		if uidCookie, err := c.Cookie("userID"); err == nil && uidCookie != "" {
			userID = uidCookie
		}
	}
	if userID == "" {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Need to Login"))
		return
	}
	updatedContent, err := h.processContentImages(content, userID, accessToken)
	if err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to process images"))
		return
	}
	content = updatedContent

	payload := map[string]interface{}{
		"title":     title,
		"content":   content,
		"published": published,
	}
	reqBody, _ := json.Marshal(payload)
	// Determine if this is a create or update based on hidden form field 'articleNumber'
	articleNumber := c.PostForm("articleNumber")
	var method, reqURL string
	if articleNumber == "" {
		method = "POST"
		reqURL = apiGatewayURL + "/api/v1/posts"
	} else {
		method = "PUT"
		reqURL = apiGatewayURL + "/api/v1/posts/" + articleNumber
	}

	req, err := http.NewRequest(method, reqURL, bytes.NewReader(reqBody))
	if err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to create request"))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to save post"))
		return
	}
	defer resp.Body.Close()
	var body Post
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to parse response"))
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errMsg map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errMsg)
		msg := ""
		if errMsg["error"] != nil {
			msg = fmt.Sprint(errMsg["error"])
		}
		if msg == "" {
			msg = "Failed to save post"
		}
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape(msg))
		return
	}
	c.Redirect(http.StatusFound, "/blog/"+strconv.FormatUint(uint64(body.ID), 10))
}

// processContentImages parses HTML content, uploads base64 images and returns updated HTML.
func (h *postHandler) processContentImages(content string, userID string, accessToken string) (string, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return content, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	// Walk nodes and collect img nodes with data URLs
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for i, a := range n.Attr {
				if a.Key == "src" {
					src := a.Val
					if strings.HasPrefix(src, "data:image/") && strings.Contains(src, ";base64,") {
						// capture node & attr index
						wg.Add(1)
						go func(node *html.Node, attrIndex int, data string) {
							defer wg.Done()
							imgURL, err := h.uploadBase64Image(data, userID, accessToken)
							if err != nil {
								mu.Lock()
								if firstErr == nil {
									firstErr = err
								}
								mu.Unlock()
								return
							}
							mu.Lock()
							node.Attr[attrIndex].Val = imgURL
							mu.Unlock()
						}(n, i, src)
					}
					break
				}
			}

		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	wg.Wait()
	if firstErr != nil {
		return "", firstErr
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type uploadImageRequest struct {
	Filename string `json:"filename"`
	UserId   string `json:"userId"`
	Data     string `json:"data"` // base64 string
}
type ImageResponse struct {
	URL  string
	Name string
	Size int64
}

// 업로드 로직 분리 (가독성 및 재사용성)
func (h *postHandler) uploadBase64Image(imgSrc string, userID string, accessToken string) (string, error) {
	ext := ".png"
	if semi := strings.Index(imgSrc, ";"); semi != -1 && len(imgSrc) > 11 {
		mime := imgSrc[11:semi]
		switch mime {
		case "jpeg", "jpg":
			ext = ".jpg"
		case "gif":
			ext = ".gif"
		case "webp":
			ext = ".webp"
		case "svg+xml":
			ext = ".svg"
		}
	}

	apiImgService := os.Getenv("IMG_SERVICE_URL")
	if apiImgService == "" {
		apiImgService = "http://img-service:8083"
	}
	// generate unique filename
	fname := fmt.Sprintf("%s-%s%s", strconv.FormatInt(time.Now().UnixNano(), 10), strconv.FormatInt(time.Now().Unix()%100000, 10), ext)

	reqBody := uploadImageRequest{
		Filename: fname,
		UserId:   userID,
		Data:     imgSrc,
	}

	bodyBytes, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error
	var resp *http.Response
	// simple retry: 2 attempts
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest("POST", apiImgService+"/blog-image", bytes.NewReader(bodyBytes))
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if accessToken != "" {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}
		resp, err = client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}
		// non-2xx
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("img service returned %d: %s", resp.StatusCode, string(b))
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
	if resp == nil {
		if lastErr == nil {
			lastErr = fmt.Errorf("failed to call img service")
		}
		return "", lastErr
	}
	defer resp.Body.Close()

	var imgResp ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgResp); err != nil {
		return "", fmt.Errorf("failed to parse image upload response: %w", err)
	}
	if imgResp.URL == "" {
		return "", fmt.Errorf("empty url from img service")
	}
	blobBase := h.cfg.BlobBaseUrl
	// fallback default for local azurite
	return blobBase + "/" + imgResp.URL, nil
}
