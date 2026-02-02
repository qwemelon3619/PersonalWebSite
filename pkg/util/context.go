package util

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// GetUserID extracts user_id from gin context
func GetUserID(c *gin.Context) (uint, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, fmt.Errorf("user_id not found in context")
	}
	if uid, ok := userID.(uint); ok {
		return uid, nil
	}
	return 0, fmt.Errorf("user_id is not uint")
}

// GetUsername extracts username from gin context
func GetUsername(c *gin.Context) (string, error) {
	username, exists := c.Get("username")
	if !exists {
		return "", fmt.Errorf("username not found in context")
	}
	if uname, ok := username.(string); ok {
		return uname, nil
	}
	return "", fmt.Errorf("username is not string")
}
