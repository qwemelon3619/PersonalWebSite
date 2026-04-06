package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	pkgconfig "seungpyo.lee/PersonalWebSite/pkg/config"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/model"
)

type stubAuthService struct {
	getUserByIDFn  func(id uint) (*domain.User, error)
	refreshTokenFn func(refreshToken string) (string, string, error)
	oAuthLoginFn   func(provider, code string) (*model.LoginResponse, *domain.GoogleUserInfo, error)
	getUserByEmail func(email string) (*domain.User, error)
}

func (s *stubAuthService) OAuthLogin(provider, code string) (*model.LoginResponse, *domain.GoogleUserInfo, error) {
	return s.oAuthLoginFn(provider, code)
}
func (s *stubAuthService) GetUserByEmail(email string) (*domain.User, error) {
	return s.getUserByEmail(email)
}
func (s *stubAuthService) GetUserByID(id uint) (*domain.User, error) {
	return s.getUserByIDFn(id)
}
func (s *stubAuthService) RefreshToken(refreshToken string) (string, string, error) {
	return s.refreshTokenFn(refreshToken)
}

func newTestHandler(svc domain.AuthService) *AuthHandler {
	cfg := &config.AuthConfig{
		GlobalConfig:       pkgconfig.GlobalConfig{RefreshTokenTTL: 60, AccessTokenTTL: 15, ServerPort: "8081"},
		MYDOMAIN:           "http://localhost:3000",
		GoogleClientID:     "cid",
		GoogleClientSecret: "csecret",
	}
	return NewAuthHandler(svc, cfg, nil)
}

func TestGetUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{
		getUserByIDFn: func(id uint) (*domain.User, error) {
			return &domain.User{ID: id, Username: "tester"}, nil
		},
	})
	r := gin.New()
	r.GET("/users/:id", h.GetUser)

	req := httptest.NewRequest(http.MethodGet, "/users/3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetUser_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{})
	r := gin.New()
	r.GET("/users/:id", h.GetUser)

	req := httptest.NewRequest(http.MethodGet, "/users/not-number", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{
		getUserByIDFn: func(id uint) (*domain.User, error) {
			return nil, errors.New("user not found")
		},
	})
	r := gin.New()
	r.GET("/users/:id", h.GetUser)

	req := httptest.NewRequest(http.MethodGet, "/users/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRefresh_UsesCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{
		refreshTokenFn: func(refreshToken string) (string, string, error) {
			if refreshToken != "cookie-refresh" {
				t.Fatalf("unexpected refresh token %q", refreshToken)
			}
			return "new-access", "new-refresh", nil
		},
	})
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "cookie-refresh"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(w.Result().Cookies()) == 0 {
		t.Fatalf("expected refresh cookie to be set")
	}
}

func TestRefresh_UsesBodyWhenCookieMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{
		refreshTokenFn: func(refreshToken string) (string, string, error) {
			if refreshToken != "body-refresh" {
				t.Fatalf("unexpected refresh token %q", refreshToken)
			}
			return "new-access", "", nil
		},
	})
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refresh_token":"body-refresh"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRefresh_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{})
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRefresh_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{
		refreshTokenFn: func(refreshToken string) (string, string, error) {
			return "", "", errors.New("invalid refresh token")
		},
	})
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refresh_token":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestOAuthGoogleLogin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origRead := readRandom
	readRandom = func(p []byte) (int, error) {
		for i := range p {
			p[i] = byte(i + 1)
		}
		return len(p), nil
	}
	t.Cleanup(func() { readRandom = origRead })

	h := newTestHandler(&stubAuthService{})
	r := gin.New()
	r.GET("/oauth/google/login", h.OAuthGoogleLogin)

	req := httptest.NewRequest(http.MethodGet, "/oauth/google/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "state=") {
		t.Fatalf("expected redirect to include state, got %q", loc)
	}
	foundStateCookie := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "oauth_state" && c.Value != "" {
			foundStateCookie = true
		}
	}
	if !foundStateCookie {
		t.Fatalf("expected oauth_state cookie to be set")
	}
}

func TestOAuthGoogleLogin_StateGenerationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origRead := readRandom
	readRandom = func(p []byte) (int, error) { return 0, errors.New("rand fail") }
	t.Cleanup(func() { readRandom = origRead })

	h := newTestHandler(&stubAuthService{})
	r := gin.New()
	r.GET("/oauth/google/login", h.OAuthGoogleLogin)

	req := httptest.NewRequest(http.MethodGet, "/oauth/google/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestOAuthGoogleCallback_MissingState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{})
	r := gin.New()
	r.GET("/oauth/google/callback", h.OAuthGoogleCallback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/google/callback?code=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOAuthGoogleCallback_MissingStateCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{})
	r := gin.New()
	r.GET("/oauth/google/callback", h.OAuthGoogleCallback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/google/callback?state=s&code=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOAuthGoogleCallback_StateMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{})
	r := gin.New()
	r.GET("/oauth/google/callback", h.OAuthGoogleCallback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/google/callback?state=s&code=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "x"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOAuthGoogleCallback_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{})
	r := gin.New()
	r.GET("/oauth/google/callback", h.OAuthGoogleCallback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/google/callback?state=s", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "s"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOAuthGoogleCallback_OAuthLoginError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{
		oAuthLoginFn: func(provider, code string) (*model.LoginResponse, *domain.GoogleUserInfo, error) {
			return nil, nil, errors.New("oauth failed")
		},
	})
	r := gin.New()
	r.GET("/oauth/google/callback", h.OAuthGoogleCallback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/google/callback?state=s&code=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "s"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestOAuthGoogleCallback_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{
		oAuthLoginFn: func(provider, code string) (*model.LoginResponse, *domain.GoogleUserInfo, error) {
			return &model.LoginResponse{
				Token:        "access",
				RefreshToken: "refresh",
				ExpiresAt:    time.Now().Add(10 * time.Minute).Unix(),
				User:         model.User{ID: 55, Username: "tester"},
			}, nil, nil
		},
	})
	r := gin.New()
	r.GET("/oauth/google/callback", h.OAuthGoogleCallback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/google/callback?state=s&code=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "s"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if w.Header().Get("Location") != "http://localhost:3000" {
		t.Fatalf("unexpected redirect location: %q", w.Header().Get("Location"))
	}

	cookies := w.Result().Cookies()
	hasAccess, hasRefresh, hasUserID, invalidatedState := false, false, false, false
	for _, c := range cookies {
		switch c.Name {
		case "access_token":
			hasAccess = true
		case "refresh_token":
			hasRefresh = true
		case "userId":
			hasUserID = true
		case "oauth_state":
			if c.MaxAge < 0 {
				invalidatedState = true
			}
		}
	}
	if !hasAccess || !hasRefresh || !hasUserID || !invalidatedState {
		t.Fatalf("expected auth cookies and invalidated oauth_state to be set")
	}
}

func TestRefresh_ResponseContainsTokenField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&stubAuthService{
		refreshTokenFn: func(refreshToken string) (string, string, error) {
			return "new-access", "", nil
		},
	})
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refresh_token":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if body["token"] != "new-access" {
		t.Fatalf("expected token in response")
	}
}
