package chat

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "github.com/gorilla/websocket"
    "github.com/golang-jwt/jwt/v5"
    "io"
    "os"
    "github.com/google/uuid"
    "log"
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

// POST /messages — Отправить (публичное или приватное)
func (h *Handler) PostMessage(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
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
        log.Printf("Failed to save message: %v", err)
        http.Error(w, "Failed to save message", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
}

// GET /messages — История публичного чата (+ attachment если есть)
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    messages, err := h.service.GetGeneralMessages(50)
    if err != nil {
        log.Printf("Failed to fetch messages: %v", err)
        http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(messages)
}

// GET /messages/{email} — Личная переписка (+ attachment если есть)
func (h *Handler) GetConversationMessages(w http.ResponseWriter, r *http.Request, otherEmail string) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
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
        log.Printf("Failed to fetch conversation: %v", err)
        http.Error(w, "Failed to fetch conversation: "+err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(messages)
}

// POST /messages/attachment — сообщение с файлом (можно без текста)
func (h *Handler) PostMessageWithAttachment(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
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

    err := r.ParseMultipartForm(10 << 20)
    if err != nil {
        http.Error(w, "Could not parse multipart form", http.StatusBadRequest)
        return
    }

    content := r.FormValue("content") // теперь может быть пустым или отсутствовать!
    // НЕ делаем проверку на присутствие контента:
    // if content == "" { ... } — этого больше нет!

    // файл обязателен!
    file, handler, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "File is required", http.StatusBadRequest)
        return
    }
    defer file.Close()

    filePath := fmt.Sprintf("storage/%d_%s", time.Now().UnixNano(), handler.Filename)
    out, err := os.Create(filePath)
    if err != nil {
        log.Printf("Failed to save file: %v", err)
        http.Error(w, "Could not save file", http.StatusInternalServerError)
        return
    }
    defer out.Close()
    if _, err = io.Copy(out, file); err != nil {
        log.Printf("Failed to save file: %v", err)
        http.Error(w, "Could not save file", http.StatusInternalServerError)
        return
    }

    messageID := uuid.NewString()
    if err := h.service.SaveMessageWithID(messageID, userID, nil, content); err != nil {
        log.Printf("Failed to save message: %v", err)
        http.Error(w, "Failed to save message", http.StatusInternalServerError)
        return
    }

    if err := h.service.SaveAttachment(messageID, userID, filePath, handler.Filename, handler.Header.Get("Content-Type")); err != nil {
        log.Printf("Failed to save attachment: %v", err)
        http.Error(w, "Failed to save attachment", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"message_id": messageID})
}

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