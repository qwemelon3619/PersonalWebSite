package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	Service domain.AuthService
	Config  *config.AuthConfig
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service domain.AuthService, cfg *config.AuthConfig) *AuthHandler {
	return &AuthHandler{Service: service, Config: cfg}
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
	// Set refresh token as secure, HttpOnly cookie with SameSite policy.
	refreshToken := resp.RefreshToken
	cookieSecure := c.Request.TLS != nil
	maxAge := int(h.Config.RefreshTokenTTL * 60) // minutes -> seconds
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		Domain:   "",
		MaxAge:   maxAge,
		Secure:   cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	c.JSON(http.StatusOK, gin.H{"token": resp.Token, "expires_at": resp.ExpiresAt, "user": resp.User})
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

// Refresh handles POST /refresh to issue a new access token using a refresh token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	// Try cookie first
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		// Try JSON body {"refresh_token":"..."}
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
			return
		}
		refreshToken = body.RefreshToken
	}

	newAccess, newRefresh, err := h.Service.RefreshToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	// Set new refresh token as cookie (rotation)
	if newRefresh != "" {
		cookieSecure := c.Request.TLS != nil
		maxAge := int(h.Config.RefreshTokenTTL * 60)
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "refresh_token",
			Value:    newRefresh,
			Path:     "/",
			Domain:   "",
			MaxAge:   maxAge,
			Secure:   cookieSecure,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
	c.JSON(http.StatusOK, gin.H{"token": newAccess})
}
