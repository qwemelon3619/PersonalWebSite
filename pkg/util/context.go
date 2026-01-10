package util

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// GetUserID extracts user_id from X-User-Id header
func GetUserID(c *gin.Context) (uint, bool) {
	userIDStr := c.GetHeader("X-User-Id")
	if userIDStr == "" {
		return 0, false
	}
	var uid uint
	_, err := fmt.Sscanf(userIDStr, "%d", &uid)
	if err != nil {
		return 0, false
	}
	return uid, true
}

// GetUsername extracts username from X-Username header
func GetUsername(c *gin.Context) (string, bool) {
	username := c.GetHeader("X-Username")
	if username == "" {
		return "", false
	}
	return username, true
}
