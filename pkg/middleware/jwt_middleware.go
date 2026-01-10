package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
)

// AuthMiddleware returns a Gin middleware that validates JWT tokens and injects claims into the context.
func AuthMiddleware(tokenManager jwt.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")
		claims, err := tokenManager.ValidateAccessToken(tokenString)
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "access token expired"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
			return
		}
		// Inject claims into context for downstream handlers
		c.Request.Header.Set("X-User-Id", fmt.Sprintf("%d", claims.UserID))
		c.Request.Header.Set("X-Username", claims.Username)
		c.Next()
	}
}
