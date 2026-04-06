package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

func TestAuthOrRefreshMiddleware_ValidToken(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			if token == "valid-token" {
				return &jwt.Claims{UserID: 7, Username: "alice"}, nil
			}
			return nil, errors.New("invalid token")
		},
	}

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, "http://127.0.0.1:65534", 15))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":  c.Request.Header.Get("X-User-Id"),
			"username": c.Request.Header.Get("X-Username"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["user_id"] != "7" {
		t.Fatalf("expected user_id=7, got %q", body["user_id"])
	}
	if body["username"] != "alice" {
		t.Fatalf("expected username=alice, got %q", body["username"])
	}
}

func TestAuthOrRefreshMiddleware_ExpiredTokenRefreshes(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			switch token {
			case "expired-token":
				return nil, jwt.ErrTokenExpired
			case "refreshed-token":
				return &jwt.Claims{UserID: 11, Username: "bob"}, nil
			default:
				return nil, errors.New("invalid token")
			}
		},
	}
	refreshToken := "refresh-token"

	authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/refresh" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req["refresh_token"] != "refresh-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "refreshed-token"})
	}))
	defer authService.Close()

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, authService.URL, 15))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":       c.Request.Header.Get("X-User-Id"),
			"username":      c.Request.Header.Get("X-Username"),
			"authorization": c.Request.Header.Get("Authorization"),
			"refreshed":     c.Request.Header.Get("X-Refreshed"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Refreshed"); got != "1" {
		t.Fatalf("expected X-Refreshed header to be 1, got %q", got)
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), "access_token=") {
		t.Fatalf("expected access_token cookie to be set, got %q", w.Header().Get("Set-Cookie"))
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if body["user_id"] != strconv.FormatUint(uint64(11), 10) {
		t.Fatalf("expected user_id=11, got %q", body["user_id"])
	}
	if body["username"] != "bob" {
		t.Fatalf("expected username=bob, got %q", body["username"])
	}
	if body["authorization"] != "Bearer refreshed-token" {
		t.Fatalf("expected Authorization header replaced, got %q", body["authorization"])
	}
	if body["refreshed"] != "1" {
		t.Fatalf("expected request X-Refreshed=1, got %q", body["refreshed"])
	}
}

func TestAuthOrRefreshMiddleware_MissingAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
	}

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, "http://example.com", 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_InvalidAuthorizationPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
	}

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, "http://example.com", 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_EmptyBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			if token == "" {
				return nil, errors.New("empty token")
			}
			return &jwt.Claims{UserID: 1, Username: "u"}, nil
		},
	}

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, "http://example.com", 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer   ")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_InvalidAccessToken_NonExpired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			return nil, errors.New("signature invalid")
		},
	}

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, "http://example.com", 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_ExpiredTokenWithoutRefreshCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) { return nil, jwt.ErrTokenExpired },
	}

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, "http://example.com", 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_StopsWhenAlreadyRefreshed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
	}

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, "http://example.com", 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-Refreshed", "1")
	req.Header.Set("Authorization", "Bearer any")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_RefreshNetworkFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) { return nil, jwt.ErrTokenExpired },
	}

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, "http://127.0.0.1:1", 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_RefreshNon200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) { return nil, jwt.ErrTokenExpired },
	}
	authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer authService.Close()

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, authService.URL, 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_InvalidRefreshResponseJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) { return nil, jwt.ErrTokenExpired },
	}
	authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer authService.Close()

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, authService.URL, 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_RefreshResponseWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) { return nil, jwt.ErrTokenExpired },
	}
	authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer authService.Close()

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, authService.URL, 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_RefreshedTokenInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			if token == "expired-token" {
				return nil, jwt.ErrTokenExpired
			}
			return nil, errors.New("still invalid")
		},
	}
	authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "bad-new-token"})
	}))
	defer authService.Close()

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, authService.URL, 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthOrRefreshMiddleware_RefreshRequestIncludesTokenBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := &stubTokenManager{
		validateAccessTokenFn: func(token string) (*jwt.Claims, error) {
			if token == "expired-token" {
				return nil, jwt.ErrTokenExpired
			}
			return &jwt.Claims{UserID: 1, Username: "u"}, nil
		},
	}

	captured := make(chan string, 1)
	authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		captured <- payload["refresh_token"]
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "new-token"})
	}))
	defer authService.Close()

	r := gin.New()
	r.Use(AuthOrRefreshMiddleware(tokenManager, authService.URL, 15))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "expected-refresh"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := <-captured; got != "expected-refresh" {
		t.Fatalf("expected refresh token in body, got %q", got)
	}
}
