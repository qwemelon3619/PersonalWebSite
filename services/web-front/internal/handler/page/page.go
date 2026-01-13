package handler

import (
	"encoding/json"
	"net/http"

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
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	apiGatewayURL := h.cfg.ApiGatewayURL
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	posts := []blogHandler.Post{}
	resp, err := http.Get(apiGatewayURL + "/api/v1/posts")
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
	c.HTML(http.StatusOK, "index.html", gin.H{
		"isLoggedIn": isLoggedIn,
		"username":   username,
		"posts":      posts,
	})
}

func (h *pageHandler) About(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	c.HTML(http.StatusOK, "about.html", gin.H{
		"isLoggedIn": isLoggedIn,
		"username":   username,
	})
}

func (h *pageHandler) Contact(c *gin.Context) {
	c.HTML(http.StatusOK, "contact.html", gin.H{})
}

func (h *pageHandler) OpenSource(c *gin.Context) {
	c.HTML(http.StatusOK, "opensource.html", gin.H{})
}

func (h *pageHandler) Error(c *gin.Context) {
	c.HTML(http.StatusOK, "error.html", gin.H{})
}
