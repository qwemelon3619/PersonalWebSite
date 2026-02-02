package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
	"seungpyo.lee/PersonalWebSite/pkg/logger"
)

var log = logger.New("debug")

// AuthOrRefreshMiddleware validates access token; if expired, it calls auth-service /refresh
// to obtain a new access token, sets it as a cookie, updates the request Authorization header,
// and injects X-User-Id/X-Username into the request headers.
func AuthOrRefreshMiddleware(tokenManager jwt.TokenManager, authServiceURL string, accessTokenTTLMinutes int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// prevent multiple refresh attempts for the same request
		log.Debug("AuthOrRefreshMiddleware invoked")
		if c.GetHeader("X-Refreshed") == "1" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token refresh failed previously"})
			return
		}
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)
		claims, err := tokenManager.ValidateAccessToken(tokenString)
		if err == nil {
			// valid
			log.Debug("access token valid")
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Request.Header.Set("X-User-Id", strconv.FormatUint(uint64(claims.UserID), 10))
			c.Request.Header.Set("X-Username", claims.Username)
			c.Next()
			return
		}
		log.Debug("access token invalid: " + err.Error())

		// If expired, try refresh
		if !errors.Is(err, jwt.ErrTokenExpired) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
			return
		}
		// get refresh token from cookie
		refreshToken, errCookie := c.Cookie("refresh_token")
		if errCookie != nil || refreshToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refresh token required"})
			return
		}
		// call auth-service /refresh
		body := map[string]string{"refresh_token": refreshToken}
		bb, _ := json.Marshal(body)
		client := &http.Client{Timeout: 3 * time.Second}
		req, _ := http.NewRequest("POST", strings.TrimRight(authServiceURL, "/")+"/refresh", bytes.NewReader(bb))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil || resp == nil {
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "failed to refresh token"})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refresh failed"})
			return
		}
		var respBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "invalid refresh response"})
			return
		}
		tokenVal, ok := respBody["token"].(string)
		if !ok || tokenVal == "" {
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "no token in refresh response"})
			return
		}
		// set cookie with new access token
		c.SetCookie("access_token", tokenVal, accessTokenTTLMinutes*60, "/", "", false, true)
		// update request header and validate to extract claims
		c.Request.Header.Set("Authorization", "Bearer "+tokenVal)
		// mark request as refreshed to avoid loops
		c.Request.Header.Set("X-Refreshed", "1")
		c.Writer.Header().Set("X-Refreshed", "1")
		newClaims, err := tokenManager.ValidateAccessToken(tokenVal)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refreshed token invalid"})
			return
		}

		// inject claims
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Request.Header.Set("X-User-Id", strconv.FormatUint(uint64(newClaims.UserID), 10))
		c.Request.Header.Set("X-Username", newClaims.Username)
		c.Next()
	}
}
