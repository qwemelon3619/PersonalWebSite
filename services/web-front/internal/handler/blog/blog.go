package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/web-front/internal/config"
)

type Post struct {
	ID          uint       `json:"id" db:"id"`
	Title       string     `json:"title" db:"title"`
	Content     string     `json:"content" db:"content"`
	AuthorID    uint       `json:"author_id" db:"author_id"`
	AuthorName  string     `json:"author_name,omitempty" db:"author_name"`
	Published   bool       `json:"published" db:"published"`
	PublishedAt *time.Time `json:"published_at,omitempty" db:"published_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
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

func (h *blogHandler) List(c *gin.Context) {
	apiGatewayURL := h.cfg.ApiGatewayURL
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	// Forward optional search query to API gateway
	searchQ := c.Query("search")
	apiURL := apiGatewayURL + "/api/v1/posts"
	if searchQ != "" {
		apiURL = apiURL + "?search=" + url.QueryEscape(searchQ)
	}
	resp, err := http.Get(apiURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to fetch posts"})
		return
	}
	defer resp.Body.Close()
	var posts []Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Invalid post data"})
		return
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
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	c.HTML(http.StatusOK, "blog-list.html", gin.H{
		"posts":       pagedPosts,
		"username":    username,
		"isLoggedIn":  isLoggedIn,
		"search":      searchQ,
		"page":        page,
		"totalPages":  totalPages,
		"pageNumbers": pageNumbers,
		"prevPage":    prevPage,
		"nextPage":    nextPage,
	})
}

func (h *blogHandler) NewPostPage(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	c.HTML(http.StatusOK, "blog-post.html", gin.H{
		"username":   username,
		"isLoggedIn": isLoggedIn,
	})
}

// EditOrNew renders the blog-post page for creating a new post or editing an existing one.
// If :articleNumber parameter is present, it will load the post for editing.
func (h *blogHandler) EditOrNew(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	articleNumber := c.Param("articleNumber")
	if articleNumber == "" {
		c.HTML(http.StatusOK, "blog-post.html", gin.H{
			"username":   username,
			"isLoggedIn": isLoggedIn,
		})
		return
	}

	apiGatewayURL := h.cfg.ApiGatewayURL

	resp, err := http.Get(apiGatewayURL + "/api/v1/posts/" + articleNumber)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to fetch post for editing"})
		return
	}
	defer resp.Body.Close()
	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Invalid post data"})
		return
	}
	c.HTML(http.StatusOK, "blog-post.html", gin.H{
		"username":      username,
		"isLoggedIn":    isLoggedIn,
		"articleNumber": articleNumber,
		"post": gin.H{
			"ID":          post.ID,
			"Title":       post.Title,
			"Content":     post.Content,
			"AuthorID":    post.AuthorID,
			"AuthorName":  post.AuthorName,
			"Published":   post.Published,
			"PublishedAt": post.PublishedAt,
			"CreatedAt":   post.CreatedAt,
			"UpdatedAt":   post.UpdatedAt,
		},
	})
}

func (h *blogHandler) Article(c *gin.Context) {
	apiGatewayURL := h.cfg.ApiGatewayURL
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	articleNumber := c.Param("articleNumber")

	resp, err := http.Get(apiGatewayURL + "/api/v1/posts/" + articleNumber)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to fetch posts"})
		return
	}
	defer resp.Body.Close()
	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Invalid post data"})
		return
	}
	contentStr := post.Content
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	c.HTML(http.StatusOK, "blog-article.html", gin.H{
		"post": gin.H{
			"ID":          post.ID,
			"Title":       post.Title,
			"Content":     contentStr,
			"AuthorID":    post.AuthorID,
			"AuthorName":  post.AuthorName,
			"Published":   post.Published,
			"PublishedAt": post.PublishedAt,
			"CreatedAt":   post.CreatedAt,
			"UpdatedAt":   post.UpdatedAt,
		},
		"username":   username,
		"isLoggedIn": isLoggedIn,
	})
}

func (h *blogHandler) RemovePage(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	articleNumber := c.Param("articleNumber")
	apiGatewayURL := h.cfg.ApiGatewayURL

	resp, err := http.Get(apiGatewayURL + "/api/v1/posts/" + articleNumber)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to fetch posts"})
		return
	}
	defer resp.Body.Close()
	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Invalid post data"})
		return
	}

	c.HTML(http.StatusOK, "blog-remove.html", gin.H{
		"username":      username,
		"isLoggedIn":    isLoggedIn,
		"articleNumber": articleNumber,
		"title":         post.Title,
	})
}
func (h *blogHandler) Remove(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	if !isLoggedIn {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{"error": "Need to Login"})
		return
	}
	apiGatewayURL := h.cfg.ApiGatewayURL
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	postID := c.Param("articleNumber")
	accessToken, err := c.Cookie("access_token")
	if err != nil || accessToken == "" {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{"error": "Need to Login"})
		return
	}
	req, err := http.NewRequest("DELETE", apiGatewayURL+"/api/v1/posts/"+postID, nil)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to delete post"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		var errMsg map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errMsg)
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": errMsg["error"]})
		return
	}
	c.Redirect(http.StatusFound, "/blog")
}
