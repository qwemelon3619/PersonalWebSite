package service

import (
	"fmt"
	"time"

	"seungpyo.lee/PersonalWebSite/pkg/jwt"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/util"
)

// authService implements domain.AuthService using a UserRepository.
type authService struct {
	repo         domain.UserRepository
	config       config.AuthConfig
	TokenManager jwt.TokenManager // JWTService can be injected here if needed
}

// NewAuthService creates a new AuthService with the given UserRepository.
func NewAuthService(repo domain.UserRepository, tokenManager jwt.TokenManager) domain.AuthService {
	return &authService{repo: repo, config: *config.LoadAuthConfig(), TokenManager: tokenManager}
}

// Login authenticates a user by username and password.
func (s *authService) Login(email string, password string) (*domain.LoginResponse, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}
	if err := util.CheckPassword(user.Password, password); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}
	// Generate access and refresh tokens
	accessToken, refreshToken, err := s.TokenManager.GenerateToken(user.ID, user.Username, time.Duration(s.config.AccessTokenTTL)*time.Minute, time.Duration(s.config.RefreshTokenTTL)*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}
	resp := &domain.LoginResponse{
		Token:        accessToken,
		ExpiresAt:    time.Now().Add(time.Duration(s.config.AccessTokenTTL) * time.Minute).Unix(),
		RefreshToken: refreshToken,
		User:         *user,
	}
	return resp, nil
}

// Register creates a new user account.
func (s *authService) Register(req domain.RegisterRequest) (*domain.User, error) {
	if user, err := s.GetUserByEmail(req.Email); err == nil && user != nil {
		return nil, fmt.Errorf("email already in use")
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user := &domain.User{
		Email:    req.Email,
		Username: req.Username,
		Password: hashedPassword,
	}
	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}
	return user, nil
}

// GetUserByEmail retrieves a user by their Email.
func (s *authService) GetUserByEmail(email string) (*domain.User, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// GetUserByID retrieves a user by their ID.
func (s *authService) GetUserByID(id uint) (*domain.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// RefreshToken generates new access/refresh tokens for the given refresh token.
func (s *authService) RefreshToken(refreshToken string) (string, string, error) {
	// Validate the provided refresh token
	claims, err := s.TokenManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}
	// Generate new tokens (rotation): issue a new refresh token and access token
	newAccess, newRefresh, err := s.TokenManager.GenerateToken(claims.UserID, claims.Username, time.Duration(s.config.AccessTokenTTL)*time.Minute, time.Duration(s.config.RefreshTokenTTL)*time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}
	// Revoke old refresh token by adding to blacklist until it would naturally expire
	// The TokenManager.RevokeToken expects the token string and TTL; let it compute TTL from token claims
	_ = s.TokenManager.RevokeToken(refreshToken, 0)
	// For the service layer we return the new access token and new refresh token together
	// Caller (handler) can decide how to deliver the refresh token (cookie).
	// We encode both separated by a pipe as a simple return (handler will parse)
	return newAccess, newRefresh, nil
}
