package jwt

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// ErrTokenExpired is returned when a token has expired.
var ErrTokenExpired = errors.New("token is expired")

// Claims defines the custom JWT claims structure.
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// TokenManager provides methods for generating, validating, and revoking JWT tokens.
type TokenManager interface {
	// accessToken, refreshToken, error
	GenerateToken(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error)
	RefreshToken(refreshToken string, accessTokenExp time.Duration) (string, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
	ValidateRefreshToken(tokenString string) (*Claims, error)
	RevokeToken(tokenString string, expiresIn time.Duration) error
	IsTokenRevoked(tokenString string) (bool, error)
}

// NewTokenManager creates a new TokenManager with the given secret key and Redis client.
func NewTokenManager(secretKey string, redisClient *redis.Client) TokenManager {
	return &tokenManager{secretKey: secretKey, redis: redisClient}
}

// NewTokenManagerWithoutRedis creates a new TokenManager that does not use Redis (useful for Gateway).
func NewTokenManagerWithoutRedis(secretKey string) TokenManager {
	return &tokenManager{secretKey: secretKey}
}

// tokenManager implements TokenManager with Redis for blacklist/refresh.
type tokenManager struct {
	secretKey string
	redis     *redis.Client
}

// GenerateToken creates a new access and refresh JWT token for a user.
func (j *tokenManager) GenerateToken(userID uint, username string, accessTokenExp, refreshTokenExp time.Duration) (string, string, error) {
	// Access Token
	accessClaims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenExp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", "", err
	}

	// Refresh Token
	refreshClaims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenExp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	refreshToken := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", "", err
	}

	return accessTokenStr, refreshTokenStr, nil
}

// GenerateAccessTokenFromRefreshToken validates the refresh token and issues a new access token only.
func (j *tokenManager) RefreshToken(refreshToken string, accessTokenExp time.Duration) (string, error) {
	claims, err := j.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", err
	}
	// make new access token
	accessClaims := Claims{
		UserID:   claims.UserID,
		Username: claims.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(accessTokenExp)),
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
		},
	}
	accessToken := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", err
	}
	return accessTokenStr, nil
}

// ValidateAccessToken parses and validates only the access token (no Redis blacklist check).
func (j *tokenManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &Claims{}, func(token *jwtlib.Token) (interface{}, error) {
		return []byte(j.secretKey), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now().UTC()) {
		return nil, ErrTokenExpired
	}
	return claims, nil
}

// ValidateRefreshToken parses and validates the refresh token and checks Redis blacklist.
func (j *tokenManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	if j.redis != nil {
		isRevoked, err := j.IsTokenRevoked(tokenString)
		if err != nil {
			return nil, err
		}
		if isRevoked {
			return nil, errors.New("refresh token is revoked")
		}
	}
	token, err := jwtlib.ParseWithClaims(tokenString, &Claims{}, func(token *jwtlib.Token) (interface{}, error) {
		return []byte(j.secretKey), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// RevokeToken stores only refresh tokens in Redis blacklist until it expires.
func (j *tokenManager) RevokeToken(tokenString string, expiresIn time.Duration) error {
	if j.redis == nil {
		return errors.New("redis client not configured")
	}
	// Revoke refresh tokens: validate the token as a refresh token first
	claims, err := j.ValidateRefreshToken(tokenString)
	if err != nil {
		return errors.New("invalid token for revocation")
	}
	// Only store in Redis until the refresh token would naturally expire
	if claims.ExpiresAt != nil {
		expiresAt := claims.ExpiresAt.Time
		ttl := time.Until(expiresAt)
		if ttl <= 0 {
			return nil // already expired
		}
		ctx := context.Background()
		if expiresIn == 0 || expiresIn > ttl {
			expiresIn = ttl
		}
		return j.redis.Set(ctx, j.redisKey(tokenString), "revoked", expiresIn).Err()
	}
	return nil
}

// IsTokenRevoked checks if the token is blacklisted in Redis.
func (j *tokenManager) IsTokenRevoked(tokenString string) (bool, error) {
	if j.redis == nil {
		return false, nil
	}
	ctx := context.Background()
	res, err := j.redis.Exists(ctx, j.redisKey(tokenString)).Result()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// redisKey generates a Redis key for a JWT token.
func (j *tokenManager) redisKey(tokenString string) string {
	return "jwt:blacklist:" + tokenString
}
