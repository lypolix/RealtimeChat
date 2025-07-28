package shared

import (
	"context"
	"crypto/rsa"
	"net/http"
	"os"
	"strings"
	"time"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
)

var (
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
)

// LoadKeys загружает RSA ключи из файлов
func LoadKeys(privateKeyPath, publicKeyPath string) error {
	// Загрузка приватного ключа
	privateBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return err
	}
	PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateBytes)
	if err != nil {
		return err
	}

	// Загрузка публичного ключа
	publicBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return err
	}
	PublicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicBytes)
	return err
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	
	// Удаляем префикс "Bearer "
	return strings.TrimPrefix(authHeader, "Bearer ")
}

// JWTMiddleware проверяет JWT токен в запросе
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractToken(r)
		if tokenString == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Проверяем метод подписи
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return PublicKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Добавляем claims в контекст
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			ctx := context.WithValue(r.Context(), "userClaims", claims)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// GenerateToken создает новый JWT токен
func GenerateToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	return token.SignedString(PrivateKey)
}