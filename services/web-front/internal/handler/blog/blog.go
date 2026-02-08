package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/web-front/internal/config"
)

type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}
type Post struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Title       string     `json:"title" gorm:"type:text;not null"`
	Content     string     `json:"content" gorm:"type:text;not null"`
	EnTitle     string     `json:"en_title,omitempty" gorm:"type:text"`
	EnContent   string     `json:"en_content,omitempty" gorm:"type:text"`
	Thumbnail   string     `json:"thumbnail,omitempty" gorm:"type:text"` // URL to thumbnail image
	AuthorID    uint       `json:"author_id" gorm:"index"`
	Author      User       `json:"author" gorm:"foreignKey:AuthorID"`
	Published   bool       `json:"published" gorm:"default:false"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Tags        []*Tag     `json:"tags,omitempty" gorm:"many2many:post_tags;constraint:OnDelete:CASCADE;"`
}
type Tag struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type BlogHandler interface {
	List(c *gin.Context)
	NewPostPage(c *gin.Context)
	Article(c *gin.Context)
	RemovePage(c *gin.Context)
	Remove(c *gin.Context)
	EditOrNew(c *gin.Context)
}

type blogHandler struct {
	cfg *config.PostConfig
}

func NewBlogHandler(cfg *config.PostConfig) BlogHandler {
	return &blogHandler{cfg: cfg}
}

// ensureImageURLs prepends base to relative image URLs in Markdown image syntax.
// It leaves absolute URLs (http..., data:...) untouched and preserves optional title text.
func ensureImageURLs(md, base string) string {
	if md == "" {
		return md
	}
	re := regexp.MustCompile(`!\[([^\]]*)\]\(\s*([^\)\s]+)[^\)]*\)`)
	return re.ReplaceAllStringFunc(md, func(m string) string {
		sub := re.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		urlPart := sub[2]
		// skip absolute URLs and data URIs
		if strings.HasPrefix(urlPart, "http") || strings.HasPrefix(urlPart, "data:") {
			return m
		}
		// join base and urlPart without duplicate slashes
		joined := base
		if strings.HasSuffix(joined, "/") && strings.HasPrefix(urlPart, "/") {
			joined = joined[:len(joined)-1] + urlPart
		} else if !strings.HasSuffix(joined, "/") && !strings.HasPrefix(urlPart, "/") {
			joined = joined + "/" + urlPart
		} else {
			joined = joined + urlPart
		}
		// replace only the first occurrence of the urlPart to preserve title/other attrs
		return strings.Replace(m, urlPart, joined, 1)
	})
}

func (h *blogHandler) List(c *gin.Context) {
	apiGatewayURL := h.cfg.ApiGatewayURL

	// Forward optional search query to API gateway
	searchQ := c.Query("search")
	tagQ := c.Query("tag")
	apiURL := apiGatewayURL + "/v1/posts"
	// Build query params for search and tag
	q := url.Values{}
	if searchQ != "" {
		q.Set("search", searchQ)
	}
	if tagQ != "" {
		q.Set("tag", tagQ)
	}
	if enc := q.Encode(); enc != "" {
		apiURL = apiURL + "?" + enc
	}
	resp, err := http.Get(apiURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to fetch posts"))
		return
	}
	defer resp.Body.Close()
	var posts []Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Invalid post data"))
		return
	}
	// Convert thumbnails to full URLs if relative
	for i := range posts {
		if posts[i].Thumbnail != "" && !strings.HasPrefix(posts[i].Thumbnail, "http") {
			posts[i].Thumbnail = h.cfg.ImageBaseURL + posts[i].Thumbnail
		}
	}
	// Fetch available tags for sidebar
	tagsURL := apiGatewayURL + "/v1/tags"
	var availableTags []Tag
	if tr, err := http.Get(tagsURL); err == nil && tr.StatusCode == http.StatusOK {
		defer tr.Body.Close()
		_ = json.NewDecoder(tr.Body).Decode(&availableTags)
	}
	// Pagination logic
	pageSize := 8
	page := 1
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
		if page < 1 {
			page = 1
		}
	}
	totalPosts := len(posts)
	totalPages := (totalPosts + pageSize - 1) / pageSize
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > totalPosts {
		start = totalPosts
	}
	if end > totalPosts {
		end = totalPosts
	}
	pagedPosts := posts[start:end]
	// pageNumbers slice
	pageNumbers := []int{}
	for i := 1; i <= totalPages; i++ {
		pageNumbers = append(pageNumbers, i)
	}
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}
	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}
	userIdStr, err := c.Cookie("userId")
	userId := uint(0)
	if err == nil && userIdStr != "" {
		if parsed, parseErr := strconv.ParseUint(userIdStr, 10, 64); parseErr == nil {
			userId = uint(parsed)
		}
	}
	isLoggedIn := false
	if _, err := c.Cookie("access_token"); err == nil {
		isLoggedIn = true
	}
	c.HTML(http.StatusOK, "blog-list.html", gin.H{
		"posts":         pagedPosts,
		"tag":           tagQ,
		"userId":        userId,
		"isLoggedIn":    isLoggedIn,
		"search":        searchQ,
		"availableTags": availableTags,
		"page":          page,
		"totalPages":    totalPages,
		"pageNumbers":   pageNumbers,
		"prevPage":      prevPage,
		"nextPage":      nextPage,
	})
}

func (h *blogHandler) NewPostPage(c *gin.Context) {
	isLoggedIn := false
	if _, err := c.Cookie("access_token"); err == nil {
		isLoggedIn = true
	}
	c.HTML(http.StatusOK, "blog-post.html", gin.H{
		"isLoggedIn": isLoggedIn,
	})
}

// EditOrNew renders the blog-post page for creating a new post or editing an existing one.
// If :articleNumber parameter is present, it will load the post for editing.
func (h *blogHandler) EditOrNew(c *gin.Context) {
	userIdStr, err := c.Cookie("userId")
	userId := uint(0)
	if err == nil && userIdStr != "" {
		if parsed, parseErr := strconv.ParseUint(userIdStr, 10, 64); parseErr == nil {
			userId = uint(parsed)
		}
	}
	isLoggedIn := false
	if _, err := c.Cookie("access_token"); err == nil {
		isLoggedIn = true
	}
	articleNumber := c.Param("articleNumber")
	if articleNumber == "" {
		c.HTML(http.StatusOK, "blog-post.html", gin.H{
			"userId":     userId,
			"isLoggedIn": isLoggedIn,
		})
		return
	}

	apiGatewayURL := h.cfg.ApiGatewayURL

	resp, err := http.Get(apiGatewayURL + "/v1/posts/" + articleNumber)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to fetch post for editing"))
		return
	}
	defer resp.Body.Close()
	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Invalid post data"))
		return
	}
	// Convert thumbnail to full URL if relative
	if post.Thumbnail != "" && !strings.HasPrefix(post.Thumbnail, "http") {
		if h.cfg.ImageBaseURL == "" {
			h.cfg.ImageBaseURL = "/data"
		}
		post.Thumbnail = h.cfg.ImageBaseURL + post.Thumbnail
	}
	// For editing, pass raw Markdown content (not processed HTML)
	c.HTML(http.StatusOK, "blog-post.html", gin.H{
		"userId":        userId,
		"isLoggedIn":    isLoggedIn,
		"articleNumber": articleNumber,
		"post": gin.H{
			"ID":          post.ID,
			"Title":       post.Title,
			"EnTitle":     post.EnTitle,
			"Content":     post.Content,   // Raw Markdown
			"EnContent":   post.EnContent, // Raw Markdown
			"Thumbnail":   post.Thumbnail,
			"AuthorID":    post.AuthorID,
			"Author":      post.Author,
			"Published":   post.Published,
			"PublishedAt": post.PublishedAt,
			"CreatedAt":   post.CreatedAt,
			"UpdatedAt":   post.UpdatedAt,
			"Tags":        post.Tags,
		},
	})
}

func (h *blogHandler) Article(c *gin.Context) {
	apiGatewayURL := h.cfg.ApiGatewayURL

	articleNumber := c.Param("articleNumber")

	resp, err := http.Get(apiGatewayURL + "/v1/posts/" + articleNumber)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to fetch posts"))
		return
	}
	defer resp.Body.Close()
	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Invalid post data"))
		return
	}
	// Convert thumbnail to full URL if relative
	if post.Thumbnail != "" && !strings.HasPrefix(post.Thumbnail, "http") {
		post.Thumbnail = h.cfg.ImageBaseURL + post.Thumbnail
	}
	// Convert stored Markdown to HTML for display
	// Keep raw markdown; rendering will be done client-side using Toast UI Viewer
	contentStr := post.Content // Korean content (raw Markdown)
	enContentStr := ""
	if post.EnContent != "" {
		enContentStr = post.EnContent // English content (raw Markdown)
	}
	//Add full url to images in content if relative
	// Use regex to find image markdown syntax ![alt](url)
	// add base URL if url is relative
	contentStr = ensureImageURLs(contentStr, h.cfg.ImageBaseURL)
	if enContentStr != "" {
		enContentStr = ensureImageURLs(enContentStr, h.cfg.ImageBaseURL)
	}

	userIdStr, err := c.Cookie("userId")
	userId := uint(0)
	if err == nil && userIdStr != "" {
		if parsed, parseErr := strconv.ParseUint(userIdStr, 10, 64); parseErr == nil {
			userId = uint(parsed)
		}
	}
	isLoggedIn := false
	if _, err := c.Cookie("access_token"); err == nil {
		isLoggedIn = true
	}
	c.HTML(http.StatusOK, "blog-article.html", gin.H{
		"post": gin.H{
			"ID":          post.ID,
			"Title":       post.Title,
			"EnTitle":     post.EnTitle,
			"Content":     contentStr,   // raw Markdown
			"EnContent":   enContentStr, // raw Markdown
			"Thumbnail":   post.Thumbnail,
			"AuthorID":    post.AuthorID,
			"Author":      post.Author,
			"Published":   post.Published,
			"PublishedAt": post.PublishedAt,
			"CreatedAt":   post.CreatedAt,
			"UpdatedAt":   post.UpdatedAt,
			"Tags":        post.Tags,
		},
		"userId":     userId,
		"isLoggedIn": isLoggedIn,
	})
}

func (h *blogHandler) RemovePage(c *gin.Context) {
	userIdStr, err := c.Cookie("userId")
	userId := uint(0)
	if err == nil && userIdStr != "" {
		if parsed, parseErr := strconv.ParseUint(userIdStr, 10, 64); parseErr == nil {
			userId = uint(parsed)
		}
	}
	isLoggedIn := false
	if _, err := c.Cookie("access_token"); err == nil {
		isLoggedIn = true
	}
	articleNumber := c.Param("articleNumber")
	apiGatewayURL := h.cfg.ApiGatewayURL

	resp, err := http.Get(apiGatewayURL + "/v1/posts/" + articleNumber)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to fetch posts"))
		return
	}
	defer resp.Body.Close()
	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Invalid post data"))
		return
	}

	c.HTML(http.StatusOK, "blog-remove.html", gin.H{
		"userId":        userId,
		"isLoggedIn":    isLoggedIn,
		"articleNumber": articleNumber,
		"title":         post.Title,
	})
}
func (h *blogHandler) Remove(c *gin.Context) {
	userId, err := c.Cookie("userId")
	isLoggedIn := err == nil && userId != ""
	if !isLoggedIn {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Need to Login"))
		return
	}
	apiGatewayURL := h.cfg.ApiGatewayURL
	postID := c.Param("articleNumber")
	accessToken, err := c.Cookie("access_token")
	if err != nil || accessToken == "" {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Need to Login"))
		return
	}
	req, err := http.NewRequest("DELETE", apiGatewayURL+"/v1/posts/"+postID, nil)
	if err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to create request"))
		return
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape("Failed to delete post"))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		var errMsg map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errMsg)
		msg := ""
		if errMsg["error"] != nil {
			msg = fmt.Sprint(errMsg["error"])
		}
		if msg == "" {
			msg = "Failed to delete post"
		}
		c.Redirect(http.StatusFound, "/error?msg="+url.QueryEscape(msg))
		return
	}
	c.Redirect(http.StatusFound, "/blog")
}
