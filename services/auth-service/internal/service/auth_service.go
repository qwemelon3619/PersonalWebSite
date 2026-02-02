package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/model"
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

// OAuthLogin handles OAuth login for providers like Google.
func (s *authService) OAuthLogin(provider, code string) (*model.LoginResponse, *domain.GoogleUserInfo, error) {
	var oauthConfig *oauth2.Config
	switch provider {
	case "google":
		oauthConfig = &oauth2.Config{
			ClientID:     s.config.GoogleClientID,
			ClientSecret: s.config.GoogleClientSecret,
			RedirectURL:  s.config.MYDOMAIN + "/api/v1/auth/oauth/google/callback",
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		}
	default:
		return nil, nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Check if user exists
	if googleUser.Email != "lspyo11@gmail.com" {
		return nil, nil, fmt.Errorf("unauthorized email")
	}
	fmt.Printf("DEBUG: Google user ID: %s, Name: %s, Email: %s\n", googleUser.ID, googleUser.Name, googleUser.Email)
	user, err := s.repo.GetByProviderID("google", googleUser.ID)
	if err != nil {
		fmt.Printf("DEBUG: GetByProviderID error: %v\n", err)
		if err.Error() == "user not found" {
			// User does not exist, create new user
			fmt.Println("DEBUG: Creating new user")
			newUser := &domain.User{
				Username:   googleUser.Name,
				Email:      googleUser.Email,
				Provider:   "google",
				ProviderID: googleUser.ID,
			}
			if err := s.repo.Create(newUser); err != nil {
				return nil, nil, fmt.Errorf("failed to create user: %w", err)
			}
			user = newUser
		} else {
			return nil, nil, fmt.Errorf("failed to get user: %w", err)
		}
	} else {
		fmt.Printf("DEBUG: Found existing user: ID=%d, Username=%s\n", user.ID, user.Username)
	}

	//generate tokens
	accessToken, refreshToken, err := s.TokenManager.GenerateToken(user.ID, user.Username, time.Duration(s.config.AccessTokenTTL)*time.Minute, time.Duration(s.config.RefreshTokenTTL)*time.Minute)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}
	respModel := &model.LoginResponse{
		Token:        accessToken,
		ExpiresAt:    time.Now().Add(time.Duration(s.config.AccessTokenTTL) * time.Minute).Unix(),
		RefreshToken: refreshToken,
		User:         *user,
	}
	return respModel, nil, nil
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
