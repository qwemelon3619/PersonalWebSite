package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
)

func TestOptionalAuthMiddleware_ValidTokenInjectsIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			if token == "valid-token" {
				return &jwt.Claims{UserID: 42, Username: "alice", TokenType: "access"}, nil
			}
			return nil, errors.New("invalid token")
		},
	}

	r := gin.New()
	r.Use(OptionalAuthMiddleware(tokenManager))
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":  c.Request.Header.Get("X-User-Id"),
			"username": c.Request.Header.Get("X-Username"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["user_id"] != "42" || body["username"] != "alice" {
		t.Fatalf("expected identity injected, got %+v", body)
	}
}

func TestOptionalAuthMiddleware_NoTokenProceedsAnonymously(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			return nil, errors.New("should not be called")
		},
	}

	r := gin.New()
	r.Use(OptionalAuthMiddleware(tokenManager))
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":  c.Request.Header.Get("X-User-Id"),
			"username": c.Request.Header.Get("X-Username"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["user_id"] != "" || body["username"] != "" {
		t.Fatalf("expected no identity for anonymous caller, got %+v", body)
	}
}

func TestOptionalAuthMiddleware_InvalidTokenProceedsAnonymously(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			return nil, errors.New("invalid token")
		},
	}

	r := gin.New()
	r.Use(OptionalAuthMiddleware(tokenManager))
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":  c.Request.Header.Get("X-User-Id"),
			"username": c.Request.Header.Get("X-Username"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["user_id"] != "" || body["username"] != "" {
		t.Fatalf("expected no identity for invalid token, got %+v", body)
	}
}
