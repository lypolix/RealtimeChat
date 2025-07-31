package models

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

// User - модель пользователя системы
type User struct {
    ID           string    `json:"id" db:"id"`
    Email        string    `json:"email" db:"email"`
    PasswordHash string    `json:"-" db:"password_hash"` // Никогда не возвращается в JSON
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// LoginRequest - модель для входящего запроса на вход
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

// RegisterRequest - модель для регистрации пользователя
type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

// AuthResponse - модель ответа с токеном (и опционально User, если нужно)
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

// Message - универсальная модель сообщения (для общего и приватных чатов)
type Message struct {
    ID              string     `json:"id" db:"id"`
    UserID          string     `json:"user_id" db:"user_id"`
    RecipientUserID *string    `json:"recipient_user_id,omitempty" db:"recipient_user_id"`
    Content         string     `json:"content" db:"content"`
    CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}


type Attachment struct {
    ID        int       `json:"id" db:"id"`
    MessageID string    `json:"message_id" db:"message_id"`
    UserID    string    `json:"user_id" db:"user_id"`
    FilePath  string    `json:"file_path" db:"file_path"`
    FileName  string    `json:"file_name" db:"file_name"`
    MimeType  string    `json:"mime_type" db:"mime_type"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
