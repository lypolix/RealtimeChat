package main

import (
	"github.com/golang-jwt/jwt/v5"

	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
	"os"

	"RealtimeChat/internal/auth"
)

var (
	signKey   *rsa.PrivateKey
	verifyKey *rsa.PublicKey
)

func loadKeys() error {
	privateBytes, err := os.ReadFile("config/private.pem")
	if err != nil {
		return fmt.Errorf("failed to read private key: %v", err)
	}
	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	publicBytes, err := os.ReadFile("config/public.pem")
	if err != nil {
		return fmt.Errorf("failed to read public key: %v", err)
	}
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(publicBytes)
	return err
}

func main() {
	// 1. Загружаем ключи
	if err := loadKeys(); err != nil {
		log.Fatalf("Failed to load keys: %v", err)
	}

	// 2. Инициализируем сервисы
	authService := auth.NewService()
	authHandler := auth.NewHandler(authService)

	// 3. Настраиваем маршруты
	http.HandleFunc("/login", authHandler.Login)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Server is running")
	})

	// 4. Запускаем сервер
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
