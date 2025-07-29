package auth

import (
	"golang.org/x/crypto/bcrypt"

	"errors"

	"RealtimeChat/internal/auth/models"
)

// Service содержит бизнес-логику аутентификации
type Service struct {
	// Добавьте здесь зависимости, например:
	// userRepo UserRepository
}

// NewService создает новый экземпляр Service
func NewService() *Service {
	return &Service{
		// Инициализация зависимостей
	}
}

// Login выполняет аутентификацию пользователя
func (s *Service) Login(email, password string) (string, error) {
	// 1. Получаем пользователя из базы данных (заглушка)
	user, err := s.getUserByEmail(email) // Этот метод нужно реализовать
	if err != nil {
		return "", err
	}

	// 2. Проверяем пароль
	if !checkPasswordHash(password, user.PasswordHash) {
		return "", errors.New("invalid credentials")
	}

	// 3. Генерируем JWT токен
	token, err := generateToken(user.ID) // Этот метод нужно реализовать
	if err != nil {
		return "", err
	}

	return token, nil
}

// Вспомогательные методы (реализуйте их)
func (s *Service) getUserByEmail(email string) (*models.User, error) {

	// Заглушка - реализуйте получение пользователя из БД
	return &models.User{
		ID:           "123",
		Email:        email,
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMy...", // Пример хеша
	}, nil
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateToken(userID string) (string, error) {
	// Реализуйте генерацию JWT токена
	return "generated.jwt.token", nil
}
