package chat

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sync"
    "time"
    "github.com/gorilla/websocket"
    "github.com/golang-jwt/jwt/v5"
    "io"
    "os"
    "github.com/google/uuid"
)

type Handler struct {
    service *Service

    clients   map[string]*websocket.Conn // userID → conn
    clientsMu sync.RWMutex               // для потокобезопасности
}

func NewHandler(service *Service) *Handler {
    return &Handler{
        service: service,
        clients: make(map[string]*websocket.Conn),
    }
}

// --- Вспомогательные методы работы с клиентами ---

// addClient: одно соединение на user, старое закрывается
func (h *Handler) addClient(userID string, conn *websocket.Conn) {
    h.clientsMu.Lock()
    defer h.clientsMu.Unlock()
    if old, exists := h.clients[userID]; exists && old != nil {
        log.Printf("Closing previous WS connection for user %s", userID)
        old.Close()
    }
    h.clients[userID] = conn
}

// removeClient: удаляет WS-соединение пользователя
func (h *Handler) removeClient(userID string) {
    h.clientsMu.Lock()
    defer h.clientsMu.Unlock()
    if _, exists := h.clients[userID]; exists {
        delete(h.clients, userID)
    }
}

// sendBroadcast: публичный broadcast всем WS-подключенным
func (h *Handler) sendBroadcast(msg map[string]interface{}) {
    h.clientsMu.RLock()
    defer h.clientsMu.RUnlock()
    for userID, conn := range h.clients {
        if err := conn.WriteJSON(msg); err != nil {
            log.Printf("Broadcast: failed for %s: %v", userID, err)
        }
    }
}

// sendToUsers: отправка приватного WS-сообщения двум ID (отправителю и получателю)
func (h *Handler) sendToUsers(userIDs []string, msg map[string]interface{}) {
    h.clientsMu.RLock()
    defer h.clientsMu.RUnlock()
    for _, userID := range userIDs {
        if conn, ok := h.clients[userID]; ok && conn != nil {
            if err := conn.WriteJSON(msg); err != nil {
                log.Printf("PrivateWS: failed for %s: %v", userID, err)
            }
        }
    }
}

// --- Handlers ---

// POST /messages (публичное или приватное текстовое)
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
        Recipient *string `json:"recipient"` // email
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    var recipientUserID *string
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
    // После сохранения — рассылаем по WS
    msgPayload := map[string]interface{}{
        "user_id":           userID,
        "recipient_user_id": recipientUserID,
        "content":           req.Content,
        "created_at":        time.Now(),
    }
    if recipientUserID == nil {
        h.sendBroadcast(msgPayload) // всем
    } else {
        h.sendToUsers([]string{userID, *recipientUserID}, msgPayload)
    }
    w.WriteHeader(http.StatusCreated)
}

// GET /messages (история публичного чата)
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

// GET /messages/{email} (история личной переписки)
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

// POST /messages/attachment — сообщение с вложением (можно лично, можно публично)
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

    content := r.FormValue("content") // Может быть пустым

    var recipientUserID *string
    recipient := r.FormValue("recipient")
    if recipient != "" {
        var id string
        err := h.service.db.QueryRowContext(
            r.Context(),
            `SELECT id FROM users WHERE email = $1`,
            recipient,
        ).Scan(&id)
        if err != nil {
            http.Error(w, "Recipient user not found", http.StatusBadRequest)
            return
        }
        recipientUserID = &id
    }

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
    if err := h.service.SaveMessageWithID(messageID, userID, recipientUserID, content); err != nil {
        log.Printf("Failed to save message: %v", err)
        http.Error(w, "Failed to save message", http.StatusInternalServerError)
        return
    }
    if err := h.service.SaveAttachment(messageID, userID, filePath, handler.Filename, handler.Header.Get("Content-Type")); err != nil {
        log.Printf("Failed to save attachment: %v", err)
        http.Error(w, "Failed to save attachment", http.StatusInternalServerError)
        return
    }

    msgPayload := map[string]interface{}{
        "user_id":           userID,
        "recipient_user_id": recipientUserID,
        "attachment": map[string]string{
            "file_name": handler.Filename,
            "file_path": filePath,
            "mime_type": handler.Header.Get("Content-Type"),
        },
        "content":    content,
        "created_at": time.Now(),
    }
    if recipientUserID == nil {
        h.sendBroadcast(msgPayload)
    } else {
        h.sendToUsers([]string{userID, *recipientUserID}, msgPayload)
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"message_id": messageID})
}

// --- WebSocket endpoint с потокобезопасным хранением клиентов ---
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
    h.addClient(userID, conn)
    defer func() {
        h.removeClient(userID)
        conn.Close()
    }()
    for {
        var msg struct {
            Content   string  `json:"content"`
            Recipient *string `json:"recipient"`
        }
        err := conn.ReadJSON(&msg)
        if err != nil {
            return
        }
        var recipientUserID *string
        if msg.Recipient != nil && *msg.Recipient != "" {
            var id string
            err := h.service.db.QueryRowContext(
                r.Context(),
                `SELECT id FROM users WHERE email = $1`,
                *msg.Recipient,
            ).Scan(&id)
            if err == nil {
                recipientUserID = &id
            }
        }
        
        if err := h.service.SaveMessage(userID, recipientUserID, msg.Content); err != nil {
            log.Printf("WebSocket: failed to save message: %v", err)
            continue
        }
        payload := map[string]interface{}{
            "user_id":           userID,
            "recipient_user_id": recipientUserID,
            "content":           msg.Content,
            "created_at":        time.Now(),
        }
        if recipientUserID == nil {
            h.sendBroadcast(payload)
        } else {
            h.sendToUsers([]string{userID, *recipientUserID}, payload)
        }
    }
}

func (h *Handler) GetUserChats(w http.ResponseWriter, r *http.Request) {
    claims, ok := r.Context().Value("userClaims").(jwt.MapClaims)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    userID := fmt.Sprintf("%v", claims["user_id"])
    chats, err := h.service.GetUserChats(userID, 100)
    if err != nil {
        http.Error(w, "Failed to fetch chats", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(chats)
}