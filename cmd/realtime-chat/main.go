// @title           RealtimeChat Service API
// @version         1.0
// @description     This is a REST API for RealtimeChat.
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

package main

import (
	_ "RealtimeChat/docs"
	"RealtimeChat/internal/auth"
	"RealtimeChat/internal/chat"
	"RealtimeChat/internal/config"
	"RealtimeChat/internal/shared"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)


func OnlineStatusUpdater(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := r.Context().Value("userClaims").(map[string]interface{}) 
        var userID string
        if ok {
            if v, ok := claims["user_id"].(string); ok {
                userID = v
            }
        }
        if userID != "" {
            if err := shared.SetUserOnline(userID); err != nil {
                log.Printf("Failed to set user online status: %v", err)
            }
        }
        next.ServeHTTP(w, r)
    })
}

func main() {
    cfg := config.MustLoad()

    db, err := shared.NewDB(cfg)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    if err := shared.LoadKeys("config/private.pem", "config/public.pem"); err != nil {
        log.Fatalf("Failed to load JWT keys: %v", err)
    }
    addr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)  
    shared.InitRedis(addr, "", 0)
    shared.RedisClient = redis.NewClient(&redis.Options{
        Addr:   addr,
        Password: "",
        DB: 0,
    })

    authService := auth.NewService(db)
    authHandler := auth.NewHandler(authService)
    chatService := chat.NewService(db)
    chatHandler := chat.NewHandler(chatService)

    http.Handle("/swagger/", httpSwagger.WrapHandler)


    http.Handle("/chats",
        shared.JWTMiddleware(
            OnlineStatusUpdater(
                http.HandlerFunc(chatHandler.GetUserChats),
            ),
        ),
    )

    http.HandleFunc("/register", authHandler.Register)
    http.HandleFunc("/login", authHandler.Login)

    http.Handle("/protected",
        shared.JWTMiddleware(
            OnlineStatusUpdater(
                http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.Write([]byte("Protected area"))
                }),
            ),
        ),
    )

    http.Handle("/messages/attachment",
        shared.JWTMiddleware(
            OnlineStatusUpdater(
                http.HandlerFunc(chatHandler.PostMessageWithAttachment),
            ),
        ),
    )

    http.Handle("/messages/",
        shared.JWTMiddleware(
            OnlineStatusUpdater(
                http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    if r.Method != http.MethodGet {
                        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
                        return
                    }
                    path := r.URL.Path
                    prefix := "/messages/"
                    if !strings.HasPrefix(path, prefix) || len(path) <= len(prefix) {
                        http.Error(w, "Email required", http.StatusBadRequest)
                        return
                    }
                    email := path[len(prefix):]
                    chatHandler.GetConversationMessages(w, r, email)
                }),
            ),
        ),
    )

    http.Handle("/messages",
        shared.JWTMiddleware(
            OnlineStatusUpdater(
                http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    switch r.Method {
                    case http.MethodPost:
                        chatHandler.PostMessage(w, r)
                    case http.MethodGet:
                        chatHandler.GetMessages(w, r)
                    default:
                        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
                    }
                }),
            ),
        ),
    )

    http.Handle("/ws",
        shared.JWTMiddleware(
            OnlineStatusUpdater(
                http.HandlerFunc(chatHandler.WebSocket),
            ),
        ),
    )

    log.Printf("Server starting on :%s", cfg.Server.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, nil))
}