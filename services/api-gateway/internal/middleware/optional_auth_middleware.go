package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
)

// OptionalAuthMiddleware propagates the caller identity upstream when a valid access
// token is present, but never blocks anonymous requests. It is intended for public
// GET routes where the downstream service decides what an anonymous caller may see.
func OptionalAuthMiddleware(tokenManager jwt.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.Next()
			return
		}
		tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if tokenString == "" {
			c.Next()
			return
		}
		claims, err := tokenManager.ValidateAccessToken(tokenString)
		if err != nil {
			// Invalid/expired token: proceed anonymously rather than aborting.
			c.Next()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Request.Header.Set("X-User-Id", strconv.FormatUint(uint64(claims.UserID), 10))
		c.Request.Header.Set("X-Username", claims.Username)
		c.Next()
	}
}
