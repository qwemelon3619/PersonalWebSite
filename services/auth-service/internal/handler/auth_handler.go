package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/model"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	Service      domain.AuthService
	Config       *config.AuthConfig
	TokenManager jwt.TokenManager
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service domain.AuthService, cfg *config.AuthConfig, tokenManager jwt.TokenManager) *AuthHandler {
	return &AuthHandler{Service: service, Config: cfg, TokenManager: tokenManager}
}

// generateState generates a random state string for OAuth.
func generateState() (string, error) {
	return "random-state", nil // Simplified for now
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

// OAuthGoogleLogin initiates Google OAuth login.
func (h *AuthHandler) OAuthGoogleLogin(c *gin.Context) {
	oauthConfig := &oauth2.Config{
		ClientID:     h.Config.GoogleClientID,
		ClientSecret: h.Config.GoogleClientSecret,
		RedirectURL:  h.Config.MYDOMAIN + "/api/v1/auth/oauth/google/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}

	// Generate random state
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusFound, url)
}

// OAuthGoogleCallback handles the Google OAuth callback.
func (h *AuthHandler) OAuthGoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code not provided"})
		return
	}

	resp, _, err := h.Service.OAuthLogin("google", code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Existing user, set cookies and redirect to home
	h.setAuthCookies(c, resp)
	c.Redirect(http.StatusFound, h.Config.MYDOMAIN)
}

// setAuthCookies sets authentication cookies for the user.
func (h *AuthHandler) setAuthCookies(c *gin.Context, resp *model.LoginResponse) {
	refreshMaxAge := int(h.Config.RefreshTokenTTL * 60)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		Path:     "/",
		Domain:   "",
		MaxAge:   refreshMaxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: 0, // not set for local development
	})

	accessMaxAge := int(h.Config.AccessTokenTTL * 60)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    resp.Token,
		Path:     "/",
		Domain:   "",
		MaxAge:   accessMaxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: 0, // not set for local development
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "userId",
		Value:    fmt.Sprintf("%d", resp.User.ID),
		Path:     "/",
		Domain:   "",
		MaxAge:   accessMaxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: 0, // not set for local development
	})
}
