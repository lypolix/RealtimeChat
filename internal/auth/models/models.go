package models

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

// User - модель пользователя системы
type User struct {
    ID           string    `json:"id" db:"id"`
    Email        string    `json:"email" db:"email"`
    PasswordHash string    `json:"-" db:"password_hash"` // Пароль никогда не должен возвращаться в JSON
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// LoginRequest - модель для входящего запроса на вход
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

// AuthResponse - модель ответа с токеном
type AuthResponse struct {
    Token string `json:"token"`
    User  User   `json:"user"`
}

// TokenClaims - кастомные claims для JWT токена
type TokenClaims struct {
    jwt.RegisteredClaims
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role,omitempty"`
}