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
    clients map[string]*websocket.Conn // userID → conn
}

func NewHandler(service *Service) *Handler {
    return &Handler{
        service: service,
        clients: make(map[string]*websocket.Conn),
    }
}

// POST /messages
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

    var req struct{ Content string }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    if err := h.service.SaveMessage(userID, req.Content); err != nil {
        http.Error(w, "Failed to save message", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
}

// GET /messages
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
    messages, err := h.service.GetMessages(50)
    if err != nil {
        http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")

    json.NewEncoder(w).Encode(messages)
}

// /ws — WebSocket endpoint
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

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
        // Сохрани в базу
        h.service.SaveMessage(userID, msg.Content)
        // Рассылаем всем подключённым
        for _, c := range h.clients {
            c.WriteJSON(map[string]interface{}{
                "user_id":    userID,
                "content":    msg.Content,
                "created_at": time.Now(),
            })
        }
    }
}
