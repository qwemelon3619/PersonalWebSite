package domain

import "time"

type User struct {
	ID        uint      `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"` // Hidden in JSON responses
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	ExpiresAt    int64  `json:"expires_at"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type UserRepository interface {
	Create(user *User) error
	GetByUsername(username string) (*User, error)
	GetByEmail(email string) (*User, error)
	GetByID(id uint) (*User, error)
	Update(user *User) error
	Delete(id uint) error
}

type AuthService interface {
	Login(email string, password string) (*LoginResponse, error)
	Register(req RegisterRequest) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id uint) (*User, error)
	RefreshToken(refreshToken string) (string, error)
}
