package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	Email    string `form:"email" json:"email" binding:"required"`
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func LoginHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

func LoginPostHandler(c *gin.Context) {
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}

	email := c.PostForm("email")
	password := c.PostForm("password")
	if email == "" || password == "" {
		fmt.Println("ID or password missing")
		c.HTML(http.StatusBadRequest, "login.html", gin.H{"error": "ID and password required"})
		return
	}

	reqBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	resp, err := http.Post(apiGatewayURL+"/api/v1/auth/login", "application/json", bytes.NewReader(reqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("Remote login failed: %v\n", err)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Login failed"})
		return
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		fmt.Printf("Failed to decode login response: %v\n", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Invalid response from auth-service"})
		return
	}
	fmt.Printf("Login response JSON: %+v\n", result)
	accessToken, ok := result["token"].(string)
	if !ok || accessToken == "" {
		fmt.Println("No token received")
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "No token received"})
		return
	}
	refreshToken, ok := result["refresh_token"].(string)
	if !ok || refreshToken == "" {
		fmt.Println("No Refresh token received")
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "No refresh token received"})
		return
	}
	user, ok := result["user"].(map[string]interface{})
	if !ok || user == nil {
		fmt.Println("No user info received")
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "No user info received"})
		return
	}
	c.SetCookie("access_token", accessToken, 3600, "/", "", false, true)
	c.SetCookie("refresh_token", refreshToken, 7200, "/", "", false, true)
	c.SetCookie("user", user["username"].(string), 7200, "/", "", false, false)
	c.Redirect(http.StatusFound, "/")
}

func RegisterHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{})
}

func RegisterPostHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("Failed to bind registration data:", err)
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error": "All fields are required and must be valid",
		})
		return
	}
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{"error": "Internal server error"})
		return
	}
	resp, err := http.Post(apiGatewayURL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
	if err != nil || resp.StatusCode != http.StatusCreated {
		fmt.Printf("Remote registration failed: %v\n", err)
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"error": "Registration failed at auth-service"})
		return
	}
	defer resp.Body.Close()
	c.Redirect(http.StatusFound, "/login")
}

func LogoutHandler(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("user", "", -1, "/", "", false, false)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}
