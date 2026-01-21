package domain

import (
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/model"
)

type User = model.User

type UserRepository interface {
	Create(user *User) error
	GetByUsername(username string) (*User, error)
	GetByEmail(email string) (*User, error)
	GetByID(id uint) (*User, error)
	Update(user *User) error
	Delete(id uint) error
}

type AuthService interface {
	Login(email string, password string) (*model.LoginResponse, error)
	Register(req model.RegisterRequest) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id uint) (*User, error)
	RefreshToken(refreshToken string) (string, string, error)
}
