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
	"log"
	
)

var (
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
)

func LoadKeys(privateKeyPath, publicKeyPath string) error {
	privateBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return err
	}
	PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateBytes)
	if err != nil {
		return err
	}

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
    parts := strings.Fields(authHeader)
    if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
        return ""
    }
    return parts[1]
}


func JWTMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Println("Authorization header:", r.Header.Get("Authorization"))
        
        tokenString := extractToken(r)
        if tokenString == "" {
            http.Error(w, "Authorization header missing", http.StatusUnauthorized)
            return
        }

        log.Println("Token:", tokenString)
        
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return PublicKey, nil
        })

        if err != nil {
            log.Printf("Token validation error: %v", err)
            http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
            return
        }

        if !token.Valid {
            log.Println("Token is invalid")
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        log.Printf("Valid token with claims: %+v", token.Claims)
        
        ctx := context.WithValue(r.Context(), "userClaims", token.Claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func GenerateToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	return token.SignedString(PrivateKey)
}