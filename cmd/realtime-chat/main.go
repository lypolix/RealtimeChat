package main


import (
	"crypto/rsa"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"fmt"
    "log"
    "net/http"
	"time"
)

var (
	signKey   *rsa.PrivateKey // Приватный ключ для подписи JWT
	verifyKey *rsa.PublicKey  // Публичный ключ для проверки
)

func loadKeys() error {
	// Загрузка приватного ключа
	privateBytes, err := os.ReadFile("config/private.pem")
	if err != nil {
		return fmt.Errorf("failed to read private key: %v", err)
	}
	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	// Загрузка публичного ключа
	publicBytes, err := os.ReadFile("config/public.pem")
	if err != nil {
		return fmt.Errorf("failed to read public key: %v", err)
	}
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(publicBytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %v", err)
	}

	return nil
}


func main() {
	loadKeys() // Загружаем ключи

	// Запускаем тест JWT
	testJWT()
    if err := loadKeys(); err != nil {
		log.Fatalf("Key loading failed: %v", err)
	}
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Сервер работает")
    })

    log.Println("Сервер запущен на http://localhost:8080")
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatalf("Ошибка сервера: %v", err)
    }
}

func testJWT() {
	// 1. Создаём токен
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user": "test",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(signKey)
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}
	fmt.Println("Signed token:", tokenString)

	// 2. Проверяем токен
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return verifyKey, nil
	})
	if err != nil {
		log.Fatalf("Failed to parse token: %v", err)
	}
	fmt.Println("Token is valid:", parsedToken.Valid)
}