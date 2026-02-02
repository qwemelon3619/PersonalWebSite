package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/web-front/internal/config"
	blogHandler "seungpyo.lee/PersonalWebSite/services/web-front/internal/handler/blog"
)

type PageHandler interface {
	Index(c *gin.Context)
	About(c *gin.Context)
	Contact(c *gin.Context)
	OpenSource(c *gin.Context)
	Error(c *gin.Context)
}

type pageHandler struct {
	cfg *config.PostConfig
}

func NewPageHandler(cfg *config.PostConfig) PageHandler {
	return &pageHandler{cfg: cfg}
}

func (h *pageHandler) Index(c *gin.Context) {
	// Handle OAuth callback
	token := c.Query("token")
	if token != "" {
		c.SetCookie("access_token", token, 3600, "/", "", true, true)
		// Redirect to clean URL
		c.Redirect(http.StatusFound, "/")
		return
	}
	isLoggedIn := false
	if _, err := c.Cookie("access_token"); err == nil {
		isLoggedIn = true
	}

	apiGatewayURL := h.cfg.ApiGatewayURL
	posts := []blogHandler.Post{}
	resp, err := http.Get(apiGatewayURL + "/v1/posts")
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var allPosts []blogHandler.Post
		if err := json.NewDecoder(resp.Body).Decode(&allPosts); err == nil && len(allPosts) > 0 {
			if len(allPosts) > 3 {
				posts = allPosts[:3]
			} else {
				posts = allPosts
			}
		}
	}
	for i := range posts {
		if posts[i].Thumbnail != "" && !strings.HasPrefix(posts[i].Thumbnail, "http") {
			posts[i].Thumbnail = h.cfg.ImageBaseURL + posts[i].Thumbnail
		}
	}
	c.HTML(http.StatusOK, "index.html", gin.H{
		"isLoggedIn": isLoggedIn,
		"posts":      posts,
	})
}

func (h *pageHandler) About(c *gin.Context) {
	isLoggedIn := false
	if _, err := c.Cookie("access_token"); err == nil {
		isLoggedIn = true
	}
	c.HTML(http.StatusOK, "about.html", gin.H{
		"isLoggedIn": isLoggedIn,
	})
}

func (h *pageHandler) Contact(c *gin.Context) {
	c.HTML(http.StatusOK, "contact.html", gin.H{})
}

func (h *pageHandler) OpenSource(c *gin.Context) {
	c.HTML(http.StatusOK, "opensource.html", gin.H{})
}

func (h *pageHandler) Error(c *gin.Context) {
	// allow passing message via query param `msg` or via context key `error`
	msg := c.Query("msg")
	if msg == "" {
		if v, ok := c.Get("error"); ok {
			if s, ok := v.(string); ok {
				msg = s
			}
		}
	}
	c.HTML(http.StatusOK, "error.html", gin.H{"error": msg})
}
