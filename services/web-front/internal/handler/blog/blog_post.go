package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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
	if title == "" {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Title is required"))
		return
	}
	published := true

	content := c.PostForm("article-content")
	if content == "" {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Content is required"))
		return
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
	// Content is now Markdown, no JSON validation needed

	// Handle thumbnail upload
	var thumbnailData string
	if file, err := c.FormFile("thumbnail"); err == nil && file != nil {
		// Read file content
		fileContent, err := file.Open()
		if err != nil {
			c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to read thumbnail file"))
			return
		}
		defer fileContent.Close()
		data, err := io.ReadAll(fileContent)
		if err != nil {
			c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to read thumbnail data"))
			return
		}
		// Convert to base64
		mimeType := file.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "image/png" // default
		}
		thumbnailData = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
	}
	// Parse comma-separated tags input and include as []string
	tagsInput := c.PostForm("tags")
	tagsPayload := []string{}
	if tagsInput != "" {
		// split by comma and trim spaces
		var tags []string
		for _, t := range strings.Split(tagsInput, ",") {
			tt := strings.TrimSpace(t)
			if tt != "" {
				tags = append(tags, tt)
			}
		}
		if len(tags) > 0 {
			tagsPayload = tags
		}
	}
	payload := map[string]interface{}{
		"title":          title,
		"tags":           tagsPayload,
		"content":        content,
		"thumbnail_data": thumbnailData,
		"published":      published,
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
