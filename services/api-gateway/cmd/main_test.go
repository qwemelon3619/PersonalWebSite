package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
	internalmw "seungpyo.lee/PersonalWebSite/services/api-gateway/internal/middleware"
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

func TestProxyTo_ForwardsPathQueryBodyHeaders(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Downstream", "1")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"path":   r.URL.Path,
			"query":  r.URL.RawQuery,
			"auth":   r.Header.Get("Authorization"),
			"method": r.Method,
			"body":   string(body),
		})
	}))
	defer downstream.Close()

	r := gin.New()
	r.POST("/v1/posts/:id", proxyTo(downstream.URL+"/posts/:id"))

	req := httptest.NewRequest(http.MethodPost, "/v1/posts/42?lang=ko", bytes.NewBufferString(`{"hello":"world"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Downstream"); got != "1" {
		t.Fatalf("expected downstream header passthrough, got %q", got)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["path"] != "/posts/42" {
		t.Fatalf("expected path /posts/42, got %q", body["path"])
	}
	if body["query"] != "lang=ko" {
		t.Fatalf("expected query lang=ko, got %q", body["query"])
	}
	if body["auth"] != "Bearer test-token" {
		t.Fatalf("expected auth header to be forwarded")
	}
	if body["method"] != http.MethodPost {
		t.Fatalf("expected method POST, got %q", body["method"])
	}
	if body["body"] != `{"hello":"world"}` {
		t.Fatalf("expected body to be forwarded, got %q", body["body"])
	}
}

func TestProxyTo_ReturnsBadGatewayWhenServiceUnavailable(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/v1/posts", proxyTo("http://127.0.0.1:1/posts"))

	req := httptest.NewRequest(http.MethodGet, "/v1/posts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestProxyTo_InvalidTargetURLReturnsInternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/v1/posts", proxyTo("://bad-target"))

	req := httptest.NewRequest(http.MethodGet, "/v1/posts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestIntegration_ExpiredAccessTokenGetsRefreshedAndProxied(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			switch token {
			case "expired-token":
				return nil, jwt.ErrTokenExpired
			case "new-token":
				return &jwt.Claims{UserID: 99, Username: "integration-user"}, nil
			default:
				return nil, errors.New("invalid token")
			}
		},
	}
	refreshToken := "refresh-token"

	authSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/refresh" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "new-token"})
	}))
	defer authSvc.Close()

	postSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/posts" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"user_id_header":  r.Header.Get("X-User-Id"),
			"username_header": r.Header.Get("X-Username"),
		})
	}))
	defer postSvc.Close()

	r := gin.New()
	authMw := internalmw.AuthOrRefreshMiddleware(tokenManager, authSvc.URL, 15)
	r.POST("/v1/posts", authMw, proxyTo(postSvc.URL+"/posts"))

	req := httptest.NewRequest(http.MethodPost, "/v1/posts", bytes.NewBufferString(`{"title":"test"}`))
	req.Header.Set("Authorization", "Bearer expired-token")
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if body["user_id_header"] != "99" {
		t.Fatalf("expected proxied X-User-Id=99, got %q", body["user_id_header"])
	}
	if body["username_header"] != "integration-user" {
		t.Fatalf("expected proxied X-Username=integration-user, got %q", body["username_header"])
	}
}

func TestRoutePolicy_PostsGetUnprotectedAndWriteMethodsProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			if token == "ok-token" {
				return &jwt.Claims{UserID: 1, Username: "u"}, nil
			}
			return nil, errors.New("invalid")
		},
	}

	postSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer postSvc.Close()

	r := gin.New()
	authMw := internalmw.AuthOrRefreshMiddleware(tokenManager, "http://127.0.0.1:65534", 15)
	r.GET("/v1/posts", proxyTo(postSvc.URL+"/posts"))
	r.POST("/v1/posts", authMw, proxyTo(postSvc.URL+"/posts"))
	r.PUT("/v1/posts/:id", authMw, proxyTo(postSvc.URL+"/posts/:id"))
	r.DELETE("/v1/posts/:id", authMw, proxyTo(postSvc.URL+"/posts/:id"))

	// GET without auth should pass
	getReq := httptest.NewRequest(http.MethodGet, "/v1/posts", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected GET /v1/posts to be unprotected (200), got %d", getW.Code)
	}

	// POST without auth should fail by middleware
	postReq := httptest.NewRequest(http.MethodPost, "/v1/posts", bytes.NewBufferString(`{}`))
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)
	if postW.Code != http.StatusUnauthorized {
		t.Fatalf("expected POST /v1/posts to be protected (401), got %d", postW.Code)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/v1/posts/1", bytes.NewBufferString(`{}`))
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusUnauthorized {
		t.Fatalf("expected PUT /v1/posts/:id to be protected (401), got %d", putW.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/v1/posts/1", nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusUnauthorized {
		t.Fatalf("expected DELETE /v1/posts/:id to be protected (401), got %d", deleteW.Code)
	}
}

func TestRoutePolicy_AuthUsersRouteProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) { return nil, errors.New("invalid") },
	}

	authSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer authSvc.Close()

	r := gin.New()
	authMw := internalmw.AuthOrRefreshMiddleware(tokenManager, "http://127.0.0.1:65534", 15)
	r.GET("/v1/auth/users/:id", authMw, proxyTo(authSvc.URL+"/users/:id"))

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/users/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected protected auth users route to return 401, got %d", w.Code)
	}
}

func TestRoutePolicy_AuthRefreshRouteUnprotected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/refresh" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "x"})
	}))
	defer authSvc.Close()

	r := gin.New()
	r.POST("/v1/auth/refresh", proxyTo(authSvc.URL+"/refresh"))

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewBufferString(`{"refresh_token":"a"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected unprotected refresh route to return 200, got %d", w.Code)
	}
}

func TestRoutePolicy_OAuthRoutesDirectProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/google/login":
			w.WriteHeader(http.StatusFound)
			w.Header().Set("Location", "https://accounts.google.com/o/oauth2/v2/auth")
		case "/oauth/google/callback":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer authSvc.Close()

	r := gin.New()
	r.GET("/v1/auth/oauth/google/login", proxyTo(authSvc.URL+"/oauth/google/login"))
	r.GET("/v1/auth/oauth/google/callback", proxyTo(authSvc.URL+"/oauth/google/callback"))

	loginReq := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/google/login", nil)
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusFound {
		t.Fatalf("expected oauth login route to proxy (302), got %d", loginW.Code)
	}

	callbackReq := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/google/callback?code=abc&state=s", nil)
	callbackW := httptest.NewRecorder()
	r.ServeHTTP(callbackW, callbackReq)
	if callbackW.Code != http.StatusOK {
		t.Fatalf("expected oauth callback route to proxy (200), got %d", callbackW.Code)
	}
}
