package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/web-front/internal/config"
)

type RegisterRequest struct {
	Email    string `form:"email" json:"email" binding:"required"`
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

type AuthHandler interface {
	Login(c *gin.Context)
	Logout(c *gin.Context)
	OAuthGoogleLogin(c *gin.Context)
	OAuthGoogleRedirect(c *gin.Context)
}

type authHandler struct {
	cfg *config.PostConfig
}

func NewAuthHandler(cfg *config.PostConfig) AuthHandler {
	return &authHandler{cfg: cfg}
}

func (h *authHandler) Login(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

func (h *authHandler) Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("user", "", -1, "/", "", false, false)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

func (h *authHandler) OAuthGoogleLogin(c *gin.Context) {
	// For browser redirect, use external URL through nginx
	oauthURL := h.cfg.MYDOMAIN + "/api/v1/auth/oauth/google/login"
	c.Redirect(http.StatusFound, oauthURL)
}
func (h *authHandler) OAuthGoogleRedirect(c *gin.Context) {
	// For browser redirect, use external URL (localhost in dev)
	oauthURL := h.cfg.MYDOMAIN + "/api/v1/auth/oauth/google/callback"
	c.Redirect(http.StatusFound, oauthURL)
}
