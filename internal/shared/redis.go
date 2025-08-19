package shared

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis(addr, password string, db int) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,    
		Password: password, 
		DB:       db,       
	})

	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Не удалось подключиться к Redis: %v", err)
	}
}

func SetUserOnline(userID string) error {
	key := "user:" + userID + ":online"
	return RedisClient.Set(context.Background(), key, "true", 60 * time.Second).Err() 
}

func IsUserOnline(userID string) (bool, error) {
	key := "user:" + userID + ":online"
	val, err := RedisClient.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return false, nil 
	}
	return val == "true", err
}

func OnlineStatusUpdater(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := r.Context().Value("userClaims").(jwt.MapClaims)
        if ok {
            var userID string
			
            switch v := claims["user_id"].(type) {
            case string:
                userID = v
            default:
                userID = ""
            }
            if userID != "" {
                if err := SetUserOnline(userID); err != nil {
                    log.Printf("Failed to set user online: %v", err)
                }
            }
        }
        next.ServeHTTP(w, r)
    })
}
