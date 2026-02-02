package model

import "time"

// User represents a user entity
type User struct {
	ID         uint      `json:"id" db:"id"`
	Username   string    `json:"username" db:"username"`
	Email      string    `json:"email" db:"email"`
	Provider   string    `json:"provider,omitempty" db:"provider"`
	ProviderID string    `json:"provider_id,omitempty" db:"provider_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// LoginResponse represents the login response payload
type LoginResponse struct {
	Token        string `json:"token"`
	ExpiresAt    int64  `json:"expires_at"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

// GoogleUserInfo represents the user info from Google.
type GoogleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}
