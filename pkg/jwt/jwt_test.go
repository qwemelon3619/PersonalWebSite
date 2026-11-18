package jwt

import (
	"errors"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestGenerateToken_AccessAndRefreshAreDistinct(t *testing.T) {
	tm := NewTokenManagerWithoutRedis(testSecret).(*tokenManager)

	access, refresh, err := tm.GenerateToken(7, "alice", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatalf("expected non-empty tokens")
	}

	accessClaims, err := tm.ValidateAccessToken(access)
	if err != nil {
		t.Fatalf("access token should validate as access: %v", err)
	}
	if accessClaims.TokenType != "access" {
		t.Fatalf("expected access typ, got %q", accessClaims.TokenType)
	}

	refreshClaims, err := tm.ValidateRefreshToken(refresh)
	if err != nil {
		t.Fatalf("refresh token should validate as refresh: %v", err)
	}
	if refreshClaims.TokenType != "refresh" {
		t.Fatalf("expected refresh typ, got %q", refreshClaims.TokenType)
	}
}

func TestValidateAccessToken_RejectsRefreshToken(t *testing.T) {
	tm := NewTokenManagerWithoutRedis(testSecret).(*tokenManager)

	_, refresh, err := tm.GenerateToken(7, "alice", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if _, err := tm.ValidateAccessToken(refresh); !errors.Is(err, ErrInvalidTokenType) {
		t.Fatalf("expected ErrInvalidTokenType for refresh-as-access, got %v", err)
	}
}

func TestValidateRefreshToken_RejectsAccessToken(t *testing.T) {
	tm := NewTokenManagerWithoutRedis(testSecret).(*tokenManager)

	access, _, err := tm.GenerateToken(7, "alice", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if _, err := tm.ValidateRefreshToken(access); !errors.Is(err, ErrInvalidTokenType) {
		t.Fatalf("expected ErrInvalidTokenType for access-as-refresh, got %v", err)
	}
}

func TestRefreshToken_MintsAccessTokenOnly(t *testing.T) {
	tm := NewTokenManagerWithoutRedis(testSecret).(*tokenManager)

	access, refresh, err := tm.GenerateToken(7, "alice", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	newAccess, err := tm.RefreshToken(refresh, time.Minute)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	claims, err := tm.ValidateAccessToken(newAccess)
	if err != nil || claims.TokenType != "access" {
		t.Fatalf("expected new access token, got claims=%v err=%v", claims, err)
	}

	// An access token must not be accepted by the refresh path.
	if _, err := tm.RefreshToken(access, time.Minute); err == nil {
		t.Fatalf("expected refresh with access token to fail")
	}
}

func TestValidateAccessToken_RejectsTypelessToken(t *testing.T) {
	tm := NewTokenManagerWithoutRedis(testSecret).(*tokenManager)

	claims := Claims{
		UserID:   7,
		Username: "legacy",
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
		},
	}
	raw, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign legacy token: %v", err)
	}
	if _, err := tm.ValidateAccessToken(raw); !errors.Is(err, ErrInvalidTokenType) {
		t.Fatalf("expected typeless token rejection, got %v", err)
	}
}

func TestValidateAccessToken_ExpirationRequired(t *testing.T) {
	tm := NewTokenManagerWithoutRedis(testSecret).(*tokenManager)

	claims := Claims{
		UserID:    7,
		Username:  "alice",
		TokenType: "access",
		RegisteredClaims: jwtlib.RegisteredClaims{
			IssuedAt: jwtlib.NewNumericDate(time.Now()),
		},
	}
	raw, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	if _, err := tm.ValidateAccessToken(raw); err == nil {
		t.Fatalf("expected token without exp to be rejected")
	}
}

func TestValidateAccessToken_RejectsWrongAlg(t *testing.T) {
	tm := NewTokenManagerWithoutRedis(testSecret).(*tokenManager)

	claims := Claims{
		UserID:    7,
		Username:  "alice",
		TokenType: "access",
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
		},
	}
	raw, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS512, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	if _, err := tm.ValidateAccessToken(raw); err == nil {
		t.Fatalf("expected non-HS256 token to be rejected")
	}
}

func TestValidateSecretStrength(t *testing.T) {
	if err := ValidateSecretStrength("short"); err == nil {
		t.Fatalf("expected short secret to be rejected")
	}
	if err := ValidateSecretStrength(testSecret); err != nil {
		t.Fatalf("expected %d-byte secret to be accepted: %v", len(testSecret), err)
	}
	if err := ValidateSecretStrength(strings.Repeat("a", 32)); err != nil {
		t.Fatalf("expected 32-byte secret to be accepted: %v", err)
	}
}

func TestRedisKey_HashesToken(t *testing.T) {
	tm := NewTokenManagerWithoutRedis(testSecret).(*tokenManager)
	token := "super-secret-token-value"

	key := tm.redisKey(token)
	if strings.Contains(key, token) {
		t.Fatalf("redis key must not contain the raw token: %q", key)
	}
	if !strings.HasPrefix(key, "jwt:blacklist:") {
		t.Fatalf("unexpected key prefix: %q", key)
	}
	if len(key) != len("jwt:blacklist:")+64 {
		t.Fatalf("expected sha256 hex suffix, got %q", key)
	}
	if again := tm.redisKey(token); again != key {
		t.Fatalf("redis key must be deterministic: %q != %q", key, again)
	}
}
