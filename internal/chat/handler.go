package chat

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "github.com/gorilla/websocket"
    "github.com/golang-jwt/jwt/v5"
)

type Handler struct {
    service *Service
    clients map[string]*websocket.Conn
}

func NewHandler(service *Service) *Handler {
    return &Handler{
        service: service,
        clients: make(map[string]*websocket.Conn),
    }
}

// Отправка сообщения в общий чат
func (h *Handler) PostMessage(w http.ResponseWriter, r *http.Request) {
    claims, ok := r.Context().Value("userClaims").(jwt.MapClaims)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    userID := fmt.Sprintf("%v", claims["user_id"])
    if userID == "" || userID == "<nil>" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        Content   string  `json:"content"`
        Recipient *string `json:"recipient"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    var recipientUserID *string = nil
    if req.Recipient != nil && *req.Recipient != "" {
        var id string
        err := h.service.db.QueryRowContext(
            r.Context(),
            `SELECT id FROM users WHERE email = $1`,
            *req.Recipient,
        ).Scan(&id)
        if err != nil {
            http.Error(w, "Recipient user not found", http.StatusBadRequest)
            return
        }
        recipientUserID = &id
    }
    if err := h.service.SaveMessage(userID, recipientUserID, req.Content); err != nil {
        http.Error(w, "Failed to save message", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
}

// Получение истории сообщений общего чата
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
    messages, err := h.service.GetGeneralMessages(50)
    if err != nil {
        http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(messages)
}

func (h *Handler) GetConversationMessages(w http.ResponseWriter, r *http.Request, otherEmail string) {
    claims, ok := r.Context().Value("userClaims").(jwt.MapClaims)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    currentUserID := fmt.Sprintf("%v", claims["user_id"])
    if currentUserID == "" || currentUserID == "<nil>" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    messages, err := h.service.GetConversationMessages(currentUserID, otherEmail, 50)
    if err != nil {
        http.Error(w, "Failed to fetch conversation: "+err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(messages)
}

// WebSocket для онлайн-чата
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
    claims, ok := r.Context().Value("userClaims").(jwt.MapClaims)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    userID := fmt.Sprintf("%v", claims["user_id"])
    if userID == "" || userID == "<nil>" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
        return
    }
    h.clients[userID] = conn
    defer func() {
        delete(h.clients, userID)
        conn.Close()
    }()
    for {
        var msg struct{ Content string }
        err := conn.ReadJSON(&msg)
        if err != nil {
            return
        }
        // Сохраняется только как публичное (recipient = nil)
        _ = h.service.SaveMessage(userID, nil, msg.Content)
        for _, c := range h.clients {
            c.WriteJSON(map[string]interface{}{
                "user_id":    userID,
                "content":    msg.Content,
                "created_at": time.Now(),
            })
        }
    }
}