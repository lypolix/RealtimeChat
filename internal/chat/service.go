package chat

import (
    "context"
    "log"
    "sync"
    "time"

    "RealtimeChat/internal/shared"
    "RealtimeChat/internal/auth/models"
    "github.com/gorilla/websocket"
)

type Service struct {
    db      *shared.DB
    clients map[string]*websocket.Conn 
    mu      sync.RWMutex               
}

func NewService(db *shared.DB) *Service {
    return &Service{
        db:      db,
        clients: make(map[string]*websocket.Conn),
    }
}


func (s *Service) AddClient(userID string, conn *websocket.Conn) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if old, exists := s.clients[userID]; exists && old != nil {
        log.Printf("WebSocket: closing old conn for user %s", userID)
        old.Close()
    }
    s.clients[userID] = conn
    log.Printf("WebSocket: connected user %s, total: %d", userID, len(s.clients))
}

func (s *Service) RemoveClient(userID string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.clients[userID]; ok {
        delete(s.clients, userID)
        log.Printf("WebSocket: removed user %s, total: %d", userID, len(s.clients))
    }
}

func (s *Service) BroadcastMessage(senderID string, recipientUserID *string, payload map[string]interface{}) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    if recipientUserID == nil {
        for userID, conn := range s.clients {
            if err := conn.WriteJSON(payload); err != nil {
                log.Printf("BroadcastMessage: failed for %s: %v", userID, err)
            }
        }
    } else {
        sendTo := []string{senderID, *recipientUserID}
        for _, userID := range sendTo {
            if conn, ok := s.clients[userID]; ok && conn != nil {
                if err := conn.WriteJSON(payload); err != nil {
                    log.Printf("BroadcastMessage: failed for %s: %v", userID, err)
                }
            }
        }
    }
}

func (s *Service) GetGeneralMessages(limit int) ([]models.MessageWithAttachment, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    query := `
        SELECT m.id, m.user_id, m.recipient_user_id, m.content, m.created_at,
               a.file_name, a.file_path, a.mime_type
        FROM messages m
        LEFT JOIN attachments a ON a.message_id = m.id
        WHERE m.recipient_user_id IS NULL
        ORDER BY m.created_at DESC
        LIMIT $1
    `
    rows, err := s.db.QueryContext(ctx, query, limit)
    if err != nil {
        log.Printf("GetGeneralMessages: query failed: %v", err)
        return nil, err
    }
    defer rows.Close()

    var messages []models.MessageWithAttachment
    for rows.Next() {
        var m models.MessageWithAttachment
        var fileName, filePath, mimeType *string
        if err := rows.Scan(
            &m.ID,
            &m.UserID,
            &m.RecipientUserID,
            &m.Content,
            &m.CreatedAt,
            &fileName,
            &filePath,
            &mimeType,
        ); err != nil {
            log.Printf("GetGeneralMessages: row scan failed: %v", err)
            return nil, err
        }
        if fileName != nil && filePath != nil && mimeType != nil {
            m.Attachment = &models.AttachmentInfo{
                FileName: *fileName,
                FilePath: *filePath,
                MimeType: *mimeType,
            }
        }
        messages = append(messages, m)
    }
    return messages, nil
}

func (s *Service) GetConversationMessages(currentUserID, otherUsername string, limit int) ([]models.MessageWithAttachment, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var otherUserID string
    err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1`, otherUsername).Scan(&otherUserID)
    if err != nil {
        log.Printf("GetConversationMessages: cannot find user %s: %v", otherUsername, err)
        return nil, err
    }

    query := `
        SELECT m.id, m.user_id, m.recipient_user_id, m.content, m.created_at,
               a.file_name, a.file_path, a.mime_type
        FROM messages m
        LEFT JOIN attachments a ON a.message_id = m.id
        WHERE 
            (m.user_id = $1 AND m.recipient_user_id = $2)
            OR
            (m.user_id = $2 AND m.recipient_user_id = $1)
        ORDER BY m.created_at DESC
        LIMIT $3
    `
    rows, err := s.db.QueryContext(ctx, query, currentUserID, otherUserID, limit)
    if err != nil {
        log.Printf("GetConversationMessages: query failed: %v", err)
        return nil, err
    }
    defer rows.Close()

    var messages []models.MessageWithAttachment
    for rows.Next() {
        var m models.MessageWithAttachment
        var fileName, filePath, mimeType *string
        if err := rows.Scan(
            &m.ID,
            &m.UserID,
            &m.RecipientUserID,
            &m.Content,
            &m.CreatedAt,
            &fileName,
            &filePath,
            &mimeType,
        ); err != nil {
            log.Printf("GetConversationMessages: row scan failed: %v", err)
            return nil, err
        }
        if fileName != nil && filePath != nil && mimeType != nil {
            m.Attachment = &models.AttachmentInfo{
                FileName: *fileName,
                FilePath: *filePath,
                MimeType: *mimeType,
            }
        }
        messages = append(messages, m)
    }
    return messages, nil
}

func (s *Service) SaveMessage(userID string, recipientUserID *string, content string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    query := `INSERT INTO messages (user_id, recipient_user_id, content) VALUES ($1, $2, $3)`
    _, err := s.db.ExecContext(ctx, query, userID, recipientUserID, content)
    if err != nil {
        log.Printf("SaveMessage: insert failed (userID=%s recipientUserID=%v content='%s'): %v", userID, recipientUserID, content, err)
    }
    return err
}

func (s *Service) SaveAttachment(messageID, userID, filePath, fileName, mimeType string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    query := `INSERT INTO attachments (message_id, user_id, file_path, file_name, mime_type) VALUES ($1, $2, $3, $4, $5)`
    _, err := s.db.ExecContext(ctx, query, messageID, userID, filePath, fileName, mimeType)
    if err != nil {
        log.Printf("SaveAttachment: insert failed (messageID=%s userID=%s file=%s): %v", messageID, userID, filePath, err)
    }
    return err
}

func (s *Service) SaveMessageWithID(messageID, userID string, recipientUserID *string, content string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    query := `INSERT INTO messages (id, user_id, recipient_user_id, content) VALUES ($1, $2, $3, $4)`
    _, err := s.db.ExecContext(ctx, query, messageID, userID, recipientUserID, content)
    if err != nil {
        log.Printf("SaveMessageWithID: insert failed (messageID=%s userID=%s recipientUserID=%v content='%s'): %v", messageID, userID, recipientUserID, content, err)
    }
    return err
}

type ChatPreview struct {
    UserID        string    `json:"user_id"`
    Email         string    `json:"email"`
    LastMessage   string    `json:"last_message"`
    LastTimestamp time.Time `json:"last_timestamp"`
}

func (s *Service) GetUserChats(currentUserID string, limit int) ([]ChatPreview, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    query := `
        SELECT u.id, u.email, COALESCE(m.content, ''), m.created_at
        FROM (
            SELECT DISTINCT 
                CASE WHEN m.user_id = $1 THEN m.recipient_user_id ELSE m.user_id END as buddy_id
            FROM messages m
            WHERE (m.user_id = $1 AND m.recipient_user_id IS NOT NULL)
               OR (m.recipient_user_id = $1)
        ) dialogs
        JOIN users u ON u.id = dialogs.buddy_id
        JOIN LATERAL (
            SELECT content, created_at 
            FROM messages 
            WHERE (user_id = $1 AND recipient_user_id = u.id)
               OR (user_id = u.id AND recipient_user_id = $1)
            ORDER BY created_at DESC LIMIT 1
        ) m ON TRUE
        ORDER BY m.created_at DESC          
        LIMIT $2
    `
    rows, err := s.db.QueryContext(ctx, query, currentUserID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var res []ChatPreview
    for rows.Next() {
        var c ChatPreview
        if err := rows.Scan(&c.UserID, &c.Email, &c.LastMessage, &c.LastTimestamp); err != nil {
            return nil, err
        }
        res = append(res, c)
    }
    return res, nil
}