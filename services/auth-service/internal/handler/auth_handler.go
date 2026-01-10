package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	Service domain.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service domain.AuthService) *AuthHandler {
	return &AuthHandler{Service: service}
}

// Register handles POST /register for user registration.
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.Service.Register(req)
	if err != nil {
		fmt.Println(err)
		if err.Error() == "email already in use" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, user)
}

// Login handles POST /login for user authentication.
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.Service.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetUser handles GET /users/:id to retrieve user info.
func (h *AuthHandler) GetUser(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	_, err := fmt.Sscanf(idParam, "%d", &id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	user, err := h.Service.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}
