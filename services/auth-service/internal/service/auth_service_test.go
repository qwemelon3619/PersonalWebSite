package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	pkgconfig "seungpyo.lee/PersonalWebSite/pkg/config"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
)

type stubUserRepo struct {
	getByEmailFn      func(email string) (*domain.User, error)
	getByIDFn         func(id uint) (*domain.User, error)
	getByProviderIDFn func(provider, providerID string) (*domain.User, error)
	createFn          func(user *domain.User) error
}

func (s *stubUserRepo) Create(user *domain.User) error {
	return s.createFn(user)
}
func (s *stubUserRepo) GetByUsername(username string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (s *stubUserRepo) GetByEmail(email string) (*domain.User, error) {
	return s.getByEmailFn(email)
}
func (s *stubUserRepo) GetByProviderID(provider, providerID string) (*domain.User, error) {
	return s.getByProviderIDFn(provider, providerID)
}
func (s *stubUserRepo) GetByID(id uint) (*domain.User, error) {
	return s.getByIDFn(id)
}
func (s *stubUserRepo) Update(user *domain.User) error { return nil }
func (s *stubUserRepo) Delete(id uint) error           { return nil }

type stubTokenManager struct {
	validateRefreshTokenFn func(token string) (*jwt.Claims, error)
	generateTokenFn        func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error)
	revokeTokenFn          func(token string, expiresIn time.Duration) error
}

func (s *stubTokenManager) GenerateToken(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
	return s.generateTokenFn(userID, username, accessTokenExp, refreshTokenExp)
}
func (s *stubTokenManager) RefreshToken(refreshToken string, accessTokenExp time.Duration) (string, error) {
	return "", errors.New("not implemented")
}
func (s *stubTokenManager) ValidateAccessToken(tokenString string) (*jwt.Claims, error) {
	return nil, errors.New("not implemented")
}
func (s *stubTokenManager) ValidateRefreshToken(tokenString string) (*jwt.Claims, error) {
	return s.validateRefreshTokenFn(tokenString)
}
func (s *stubTokenManager) RevokeToken(tokenString string, expiresIn time.Duration) error {
	return s.revokeTokenFn(tokenString, expiresIn)
}
func (s *stubTokenManager) IsTokenRevoked(tokenString string) (bool, error) {
	return false, nil
}

func newServiceForTest(repo domain.UserRepository, tm jwt.TokenManager) *authService {
	return &authService{
		repo: repo,
		config: config.AuthConfig{
			GlobalConfig:       pkgconfig.GlobalConfig{AccessTokenTTL: 15, RefreshTokenTTL: 60, ServerPort: "8081"},
			GoogleClientID:     "cid",
			GoogleClientSecret: "csecret",
			MYDOMAIN:           "http://localhost:3000",
		},
		TokenManager: tm,
	}
}

func TestGetUserByEmail(t *testing.T) {
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn: func(email string) (*domain.User, error) {
			return &domain.User{Email: email, Username: "u"}, nil
		},
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	user, err := svc.GetUserByEmail("a@b.com")
	if err != nil || user.Email != "a@b.com" {
		t.Fatalf("expected success, got user=%v err=%v", user, err)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, errors.New("db err") },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, err := svc.GetUserByEmail("a@b.com")
	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected user not found error, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return &domain.User{ID: id}, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	user, err := svc.GetUserByID(4)
	if err != nil || user.ID != 4 {
		t.Fatalf("expected success, got user=%v err=%v", user, err)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, errors.New("not found") },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, err := svc.GetUserByID(4)
	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected user not found error, got %v", err)
	}
}

func TestRefreshToken_ValidateFail(t *testing.T) {
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, errors.New("bad refresh") },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, _, err := svc.RefreshToken("bad")
	if err == nil || !strings.Contains(err.Error(), "invalid refresh token") {
		t.Fatalf("expected invalid refresh token error, got %v", err)
	}
}

func TestRefreshToken_GenerateFail(t *testing.T) {
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) {
			return &jwt.Claims{UserID: 1, Username: "u"}, nil
		},
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", errors.New("gen fail")
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, _, err := svc.RefreshToken("ok")
	if err == nil || !strings.Contains(err.Error(), "failed to generate tokens") {
		t.Fatalf("expected generate error, got %v", err)
	}
}

func TestRefreshToken_SuccessAndRevokeCalled(t *testing.T) {
	revokeCalled := false
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) {
			return &jwt.Claims{UserID: 2, Username: "john"}, nil
		},
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "new-access", "new-refresh", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error {
			revokeCalled = true
			return nil
		},
	})
	access, refresh, err := svc.RefreshToken("old-refresh")
	if err != nil || access != "new-access" || refresh != "new-refresh" {
		t.Fatalf("expected success, got access=%q refresh=%q err=%v", access, refresh, err)
	}
	if !revokeCalled {
		t.Fatalf("expected revoke to be called")
	}
}

func TestOAuthLogin_UnsupportedProvider(t *testing.T) {
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, _, err := svc.OAuthLogin("github", "code")
	if err == nil || !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}

func TestOAuthLogin_ExchangeFail(t *testing.T) {
	origExchange := exchangeCode
	origFetch := fetchGoogleUser
	exchangeCode = func(oauthConfig *oauth2.Config, code string) (*oauth2.Token, error) {
		return nil, errors.New("exchange fail")
	}
	fetchGoogleUser = origFetch
	t.Cleanup(func() {
		exchangeCode = origExchange
		fetchGoogleUser = origFetch
	})

	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, _, err := svc.OAuthLogin("google", "code")
	if err == nil || !strings.Contains(err.Error(), "failed to exchange code") {
		t.Fatalf("expected exchange error, got %v", err)
	}
}

func TestOAuthLogin_FetchFail(t *testing.T) {
	origExchange := exchangeCode
	origFetch := fetchGoogleUser
	exchangeCode = func(oauthConfig *oauth2.Config, code string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "x"}, nil
	}
	fetchGoogleUser = func(oauthConfig *oauth2.Config, token *oauth2.Token) (*googleUser, error) {
		return nil, errors.New("failed to get user info: boom")
	}
	t.Cleanup(func() {
		exchangeCode = origExchange
		fetchGoogleUser = origFetch
	})

	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, _, err := svc.OAuthLogin("google", "code")
	if err == nil || !strings.Contains(err.Error(), "failed to get user info") {
		t.Fatalf("expected fetch error, got %v", err)
	}
}

func TestOAuthLogin_DecodeFail(t *testing.T) {
	origExchange := exchangeCode
	origFetch := fetchGoogleUser
	exchangeCode = func(oauthConfig *oauth2.Config, code string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "x"}, nil
	}
	fetchGoogleUser = func(oauthConfig *oauth2.Config, token *oauth2.Token) (*googleUser, error) {
		return nil, errors.New("failed to decode user info: bad json")
	}
	t.Cleanup(func() {
		exchangeCode = origExchange
		fetchGoogleUser = origFetch
	})

	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, _, err := svc.OAuthLogin("google", "code")
	if err == nil || !strings.Contains(err.Error(), "failed to decode user info") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestOAuthLogin_UnauthorizedEmail(t *testing.T) {
	origExchange := exchangeCode
	origFetch := fetchGoogleUser
	exchangeCode = func(oauthConfig *oauth2.Config, code string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "x"}, nil
	}
	fetchGoogleUser = func(oauthConfig *oauth2.Config, token *oauth2.Token) (*googleUser, error) {
		return &googleUser{ID: "gid", Email: "other@example.com", Name: "Other"}, nil
	}
	t.Cleanup(func() {
		exchangeCode = origExchange
		fetchGoogleUser = origFetch
	})

	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn:      func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:         func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) { return nil, nil },
		createFn:          func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, _, err := svc.OAuthLogin("google", "code")
	if err == nil || !strings.Contains(err.Error(), "unauthorized email") {
		t.Fatalf("expected unauthorized email error, got %v", err)
	}
}

func TestOAuthLogin_ExistingUser(t *testing.T) {
	origExchange := exchangeCode
	origFetch := fetchGoogleUser
	exchangeCode = func(oauthConfig *oauth2.Config, code string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "x"}, nil
	}
	fetchGoogleUser = func(oauthConfig *oauth2.Config, token *oauth2.Token) (*googleUser, error) {
		return &googleUser{ID: "gid", Email: "lspyo11@gmail.com", Name: "Lee"}, nil
	}
	t.Cleanup(func() {
		exchangeCode = origExchange
		fetchGoogleUser = origFetch
	})

	createCalled := false
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn: func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:    func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) {
			return &domain.User{ID: 10, Username: "existing", Email: "lspyo11@gmail.com"}, nil
		},
		createFn: func(user *domain.User) error {
			createCalled = true
			return nil
		},
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "access", "refresh", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	resp, _, err := svc.OAuthLogin("google", "code")
	if err != nil || resp == nil || resp.Token == "" {
		t.Fatalf("expected success, got resp=%v err=%v", resp, err)
	}
	if createCalled {
		t.Fatalf("did not expect create for existing user")
	}
}

func TestOAuthLogin_NewUser(t *testing.T) {
	origExchange := exchangeCode
	origFetch := fetchGoogleUser
	exchangeCode = func(oauthConfig *oauth2.Config, code string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "x"}, nil
	}
	fetchGoogleUser = func(oauthConfig *oauth2.Config, token *oauth2.Token) (*googleUser, error) {
		return &googleUser{ID: "gid", Email: "lspyo11@gmail.com", Name: "Newbie"}, nil
	}
	t.Cleanup(func() {
		exchangeCode = origExchange
		fetchGoogleUser = origFetch
	})

	createCalled := false
	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn: func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:    func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) {
			return nil, errors.New("user not found")
		},
		createFn: func(user *domain.User) error {
			createCalled = true
			user.ID = 33
			return nil
		},
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "access", "refresh", nil
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	resp, _, err := svc.OAuthLogin("google", "code")
	if err != nil || resp == nil || resp.User.ID != 33 {
		t.Fatalf("expected created user success, got resp=%v err=%v", resp, err)
	}
	if !createCalled {
		t.Fatalf("expected create for new user")
	}
}

func TestOAuthLogin_GenerateTokenFail(t *testing.T) {
	origExchange := exchangeCode
	origFetch := fetchGoogleUser
	exchangeCode = func(oauthConfig *oauth2.Config, code string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "x"}, nil
	}
	fetchGoogleUser = func(oauthConfig *oauth2.Config, token *oauth2.Token) (*googleUser, error) {
		return &googleUser{ID: "gid", Email: "lspyo11@gmail.com", Name: "Lee"}, nil
	}
	t.Cleanup(func() {
		exchangeCode = origExchange
		fetchGoogleUser = origFetch
	})

	svc := newServiceForTest(&stubUserRepo{
		getByEmailFn: func(email string) (*domain.User, error) { return nil, nil },
		getByIDFn:    func(id uint) (*domain.User, error) { return nil, nil },
		getByProviderIDFn: func(provider, providerID string) (*domain.User, error) {
			return &domain.User{ID: 10, Username: "existing", Email: "lspyo11@gmail.com"}, nil
		},
		createFn: func(user *domain.User) error { return nil },
	}, &stubTokenManager{
		validateRefreshTokenFn: func(token string) (*jwt.Claims, error) { return nil, nil },
		generateTokenFn: func(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
			return "", "", errors.New("gen fail")
		},
		revokeTokenFn: func(token string, expiresIn time.Duration) error { return nil },
	})
	_, _, err := svc.OAuthLogin("google", "code")
	if err == nil || !strings.Contains(err.Error(), "failed to generate tokens") {
		t.Fatalf("expected token generate error, got %v", err)
	}
}
