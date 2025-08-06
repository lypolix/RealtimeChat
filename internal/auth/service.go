package auth

import (
    "context"
    "errors"
    "time"

    "RealtimeChat/internal/auth/models"
    "RealtimeChat/internal/shared"
    "golang.org/x/crypto/bcrypt"
)

type Service struct {
    db *shared.DB
}

func NewService(db *shared.DB) *Service {
    return &Service{db: db}
}

func (s *Service) Login(email, password string) (string, error) {
    user, err := s.getUserByEmail(email)
    if err != nil {
        return "", err
    }

    if !checkPasswordHash(password, user.PasswordHash) {
        return "", errors.New("invalid credentials")
    }

    token, err := shared.GenerateToken(user.ID)
    if err != nil {
        return "", err
    }

    return token, nil
}

func (s *Service) createUser(user *models.User) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    query := `INSERT INTO users (email, password_hash) VALUES ($1, $2)`
    _, err := s.db.ExecContext(ctx, query, user.Email, user.PasswordHash)
    return err
}

func (s *Service) Register(email, password string) error {

    _, err := s.getUserByEmail(email)
    if err == nil {
        return errors.New("user already exists")
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }

    user := &models.User{
        Email:        email,
        PasswordHash: string(hash),
    }
    return s.createUser(user)
}

func (s *Service) getUserByEmail(email string) (*models.User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    query := `SELECT id, email, password_hash FROM users WHERE email = $1`
    row := s.db.QueryRowContext(ctx, query, email)

    var user models.User
    if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash); err != nil {
        return nil, err
    }

    return &user, nil
}

func checkPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
