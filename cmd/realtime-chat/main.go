package main

import (
    "log"
    "net/http"

    "RealtimeChat/internal/config"
    "RealtimeChat/internal/auth"
    "RealtimeChat/internal/chat"
    "RealtimeChat/internal/shared"
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
    

    // Маршруты для регистрации и входа
    http.HandleFunc("/register", authHandler.Register)
    http.HandleFunc("/login", authHandler.Login)

    // Пример защищённого эндпоинта
    http.Handle("/protected", shared.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Protected area"))
    })))

    // REST обработка сообщений (оба метода на одном маршруте)
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

    // WebSocket эндпоинт (авторизация обязательна)
    http.Handle("/ws", shared.JWTMiddleware(http.HandlerFunc(chatHandler.WebSocket)))

    // Запуск сервера
    log.Printf("Server starting on :%s", cfg.Server.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, nil))
}
 
