package handler

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func IndexHandler(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	// 최신 블로그 글 3개 가져오기
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	posts := []Post{}
	resp, err := http.Get(apiGatewayURL + "/api/v1/posts")
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var allPosts []Post
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

func AboutHandler(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	c.HTML(http.StatusOK, "about.html", gin.H{
		"isLoggedIn": isLoggedIn,
		"username":   username,
	})
}

func ContactHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "contact.html", gin.H{})
}

func OpenSourceHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "opensource.html", gin.H{})
}

func ErrorHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "error.html", gin.H{})
}

func RedirectHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "redirect.html", gin.H{})
}
