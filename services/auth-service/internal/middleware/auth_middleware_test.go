package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
)

type stubTokenManager struct {
	validateAccessTokenFn func(token string) (*jwt.Claims, error)
}

func (s *stubTokenManager) GenerateToken(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
	return "", "", errors.New("not implemented")
}
func (s *stubTokenManager) RefreshToken(refreshToken string, accessTokenExp time.Duration) (string, error) {
	return "", errors.New("not implemented")
}
func (s *stubTokenManager) ValidateAccessToken(tokenString string) (*jwt.Claims, error) {
	return s.validateAccessTokenFn(tokenString)
}
func (s *stubTokenManager) ValidateRefreshToken(tokenString string) (*jwt.Claims, error) {
	return nil, errors.New("not implemented")
}
func (s *stubTokenManager) RevokeToken(tokenString string, expiresIn time.Duration) error {
	return errors.New("not implemented")
}
func (s *stubTokenManager) IsTokenRevoked(tokenString string) (bool, error) {
	return false, errors.New("not implemented")
}

func newRouter(tm jwt.TokenManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users/:id", RequireSelf(tm), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestRequireSelf_MissingToken(t *testing.T) {
	tm := &stubTokenManager{validateAccessTokenFn: func(string) (*jwt.Claims, error) {
		return nil, errors.New("should not be called")
	}}
	w := httptest.NewRecorder()
	newRouter(tm).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/3", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireSelf_InvalidToken(t *testing.T) {
	tm := &stubTokenManager{validateAccessTokenFn: func(string) (*jwt.Claims, error) {
		return nil, errors.New("bad signature")
	}}
	req := httptest.NewRequest(http.MethodGet, "/users/3", nil)
	req.Header.Set("Authorization", "Bearer bad")
	w := httptest.NewRecorder()
	newRouter(tm).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireSelf_OtherUserForbidden(t *testing.T) {
	tm := &stubTokenManager{validateAccessTokenFn: func(string) (*jwt.Claims, error) {
		return &jwt.Claims{UserID: 7, Username: "alice", TokenType: "access"}, nil
	}}
	req := httptest.NewRequest(http.MethodGet, "/users/3", nil)
	req.Header.Set("Authorization", "Bearer good")
	w := httptest.NewRecorder()
	newRouter(tm).ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRequireSelf_OwnUserAllowed(t *testing.T) {
	tm := &stubTokenManager{validateAccessTokenFn: func(string) (*jwt.Claims, error) {
		return &jwt.Claims{UserID: 7, Username: "alice", TokenType: "access"}, nil
	}}
	req := httptest.NewRequest(http.MethodGet, "/users/7", nil)
	req.Header.Set("Authorization", "Bearer good")
	w := httptest.NewRecorder()
	newRouter(tm).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireSelf_NonNumericID(t *testing.T) {
	tm := &stubTokenManager{validateAccessTokenFn: func(string) (*jwt.Claims, error) {
		return &jwt.Claims{UserID: 7, Username: "alice", TokenType: "access"}, nil
	}}
	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	req.Header.Set("Authorization", "Bearer good")
	w := httptest.NewRecorder()
	newRouter(tm).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
