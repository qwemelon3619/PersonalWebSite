package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
	"seungpyo.lee/PersonalWebSite/services/api-gateway/internal/config"
	internalmw "seungpyo.lee/PersonalWebSite/services/api-gateway/internal/middleware"
)

func main() {

	conf := config.LoadGatewayConfig()

	TokenManager := jwt.NewTokenManagerWithoutRedis(conf.JWTSecretKey)
	r := gin.Default()
	// Auth Service proxy
	authMw := internalmw.AuthOrRefreshMiddleware(TokenManager, conf.AuthServiceURL, conf.AccessTokenTTL)
	r.POST("/v1/auth/refresh", proxyTo(conf.AuthServiceURL+"/refresh"))
	r.GET("/v1/auth/oauth/google/login", proxyTo(conf.AuthServiceURL+"/oauth/google/login"))
	r.GET("/v1/auth/oauth/google/callback", proxyTo(conf.AuthServiceURL+"/oauth/google/callback"))
	r.GET("/v1/auth/users/:id", authMw, proxyTo(conf.AuthServiceURL+"/users/:id"))
	r.PUT("/v1/auth/users/:id", authMw, proxyTo(conf.AuthServiceURL+"/users/:id"))

	// Post Service proxy
	r.GET("/v1/posts", proxyTo(conf.PostServiceURL+"/posts"))
	r.GET("/v1/posts/:id", proxyTo(conf.PostServiceURL+"/posts/:id"))
	r.GET("/v1/tags", proxyTo(conf.PostServiceURL+"/tags"))
	// Use API-gateway specific middleware that will attempt refresh on expired tokens
	r.POST("/v1/posts", authMw, proxyTo(conf.PostServiceURL+"/posts"))
	r.PUT("/v1/posts/:id", authMw, proxyTo(conf.PostServiceURL+"/posts/:id"))
	r.DELETE("/v1/posts/:id", authMw, proxyTo(conf.PostServiceURL+"/posts/:id"))

	// Img Service proxy
	// r.POST("/v1/images", authMw, proxyTo(conf.ImgServiceURL+"/blog-image"))
	// r.DELETE("/v1/images", authMw, proxyTo(conf.ImgServiceURL+"/blog-image"))

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

		// Preserve original query string when proxying
		targetURL := url
		if c.Request.URL != nil && c.Request.URL.RawQuery != "" {
			if strings.Contains(targetURL, "?") {
				targetURL = targetURL + "&" + c.Request.URL.RawQuery
			} else {
				targetURL = targetURL + "?" + c.Request.URL.RawQuery
			}
		}

		req, err := http.NewRequest(c.Request.Method, targetURL, body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "proxy request error"})
			return
		}
		req.Header = c.Request.Header.Clone()

		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
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
