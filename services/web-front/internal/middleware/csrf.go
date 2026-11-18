package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// SameOriginGuard is a lightweight CSRF defence for state-changing requests.
// For POST/PUT/DELETE (and PATCH) it rejects requests that browsers explicitly
// mark as cross-site:
//   - an Origin header whose host differs from the request Host -> 403
//   - a Sec-Fetch-Site header that is neither "same-origin" nor "none" -> 403
//
// Missing headers are allowed (legacy clients, curl, server-to-server).
func SameOriginGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
			if origin := c.GetHeader("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || !strings.EqualFold(u.Host, c.Request.Host) {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
			if site := c.GetHeader("Sec-Fetch-Site"); site != "" {
				if site != "same-origin" && site != "none" {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
		}
		c.Next()
	}
}
