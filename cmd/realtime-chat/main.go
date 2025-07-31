package main

import (
    "log"
    "net/http"

    "RealtimeChat/internal/auth"
    "RealtimeChat/internal/chat"
    "RealtimeChat/internal/config"
    "RealtimeChat/internal/shared"
    "strings"
)

func main() {

    // Загрузка конфигурации
    cfg := config.MustLoad()

    // Инициализация БД
    db, err := shared.NewDB(cfg)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Загрузка JWT ключей
    if err := shared.LoadKeys("config/private.pem", "config/public.pem"); err != nil {
        log.Fatalf("Failed to load JWT keys: %v", err)
    }

    // Инициализация сервисов
    authService := auth.NewService(db)
    authHandler := auth.NewHandler(authService)
    chatService := chat.NewService(db)
    chatHandler := chat.NewHandler(chatService)

    // Маршруты для регистрации и логина
    http.HandleFunc("/register", authHandler.Register)
    http.HandleFunc("/login", authHandler.Login)

    // Пример защищённого эндпоинта
    http.Handle("/protected", shared.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Protected area"))
    })))

    // REST обработка сообщений (POST и GET на одном маршруте, JWT обязателен)
    http.Handle("/messages", shared.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodPost:
            chatHandler.PostMessage(w, r)
        case http.MethodGet:
            chatHandler.GetMessages(w, r)
        default:
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    })))

    http.Handle("/messages/", shared.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Только GET! (POST сюда не попадёт)
        if r.Method != http.MethodGet {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        // Парсим email из пути
        path := r.URL.Path
        prefix := "/messages/"
        if !strings.HasPrefix(path, prefix) || len(path) <= len(prefix) {
            http.Error(w, "Email required", http.StatusBadRequest)
            return
        }
        email := path[len(prefix):]
        chatHandler.GetConversationMessages(w, r, email)
    })))

    http.Handle("/messages/attachment", shared.JWTMiddleware(http.HandlerFunc(chatHandler.PostMessageWithAttachment)))

    // WebSocket эндпоинт (JWT обязателен)
    http.Handle("/ws", shared.JWTMiddleware(http.HandlerFunc(chatHandler.WebSocket)))

    // Запуск HTTP-сервера
    log.Printf("Server starting on :%s", cfg.Server.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, nil))
}
