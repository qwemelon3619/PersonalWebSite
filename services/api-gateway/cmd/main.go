package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
	"seungpyo.lee/PersonalWebSite/pkg/middleware"
	"seungpyo.lee/PersonalWebSite/services/api-gateway/internal/config"
)

func main() {

	conf := config.LoadGatewayConfig()

	TokenManager := jwt.NewTokenManagerWithoutRedis(conf.JWTSecretKey)
	r := gin.Default()
	// Auth Service proxy
	r.POST("/api/v1/auth/login", proxyTo(conf.AuthServiceURL+"/login"))
	r.POST("/api/v1/auth/register", proxyTo(conf.AuthServiceURL+"/register"))

	// Post Service proxy
	r.GET("/api/v1/posts", proxyTo(conf.PostServiceURL+"/posts"))
	r.GET("/api/v1/posts/:id", proxyTo(conf.PostServiceURL+"/posts/:id"))
	r.POST("/api/v1/posts", middleware.AuthMiddleware(TokenManager), proxyTo(conf.PostServiceURL+"/posts"))
	r.PUT("/api/v1/posts/:id", middleware.AuthMiddleware(TokenManager), proxyTo(conf.PostServiceURL+"/posts/:id"))
	r.DELETE("/api/v1/posts/:id", middleware.AuthMiddleware(TokenManager), proxyTo(conf.PostServiceURL+"/posts/:id"))

	// Image Service proxy - internal use only

	log.Printf("API Gateway running on :%s", conf.ServerPort)
	if err := r.Run(":" + conf.ServerPort); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

// proxyTo creates a Gin handler that proxies requests to the specified target URL.
func proxyTo(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build the target URL with path parameters
		url := target
		for _, param := range c.Params {
			url = strings.ReplaceAll(url, ":"+param.Key, param.Value)
		}

		var body io.Reader
		if c.Request.Body != nil {
			data, _ := io.ReadAll(c.Request.Body)
			body = io.NopCloser(strings.NewReader(string(data)))
			c.Request.Body = io.NopCloser(strings.NewReader(string(data)))
		}

		req, err := http.NewRequest(c.Request.Method, url, body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "proxy request error"})
			return
		}
		req.Header = c.Request.Header.Clone()

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "service unavailable"})
			return
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			for _, vv := range v {
				c.Writer.Header().Add(k, vv)
			}
		}
		c.Writer.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(c.Writer, resp.Body)
	}
}
