package chat

import (
    "context"
    "log"
    "time"

    "RealtimeChat/internal/shared"
    "RealtimeChat/internal/auth/models"
)

type Service struct {
    db *shared.DB
}

func NewService(db *shared.DB) *Service {
    return &Service{db: db}
}

// GetGeneralMessages возвращает все публичные сообщения (recipient_user_id IS NULL) с вложением, если есть
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

// GetConversationMessages — личка между двумя пользователями (в обе стороны), с вложениями (если есть)
func (s *Service) GetConversationMessages(currentUserID, otherUsername string, limit int) ([]models.MessageWithAttachment, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Получаем id другого пользователя по email
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

// SaveMessage — сохраняет сообщение (публичное или приватное)
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

// SaveAttachment — сохраняет запись о файле (привязка к сообщению)
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

// SaveMessageWithID — сохраняет сообщение с заданным message_id (для вложения), c поддержкой recipientUserID
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