package chat

import (
    "context"
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

// GetGeneralMessages возвращает все сообщения общего чата (recipient_user_id IS NULL)
func (s *Service) GetGeneralMessages(limit int) ([]models.Message, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    query := `
        SELECT id, user_id, recipient_user_id, content, created_at 
        FROM messages 
        WHERE recipient_user_id IS NULL 
        ORDER BY created_at DESC 
        LIMIT $1
    `
    rows, err := s.db.QueryContext(ctx, query, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var messages []models.Message
    for rows.Next() {
        var m models.Message
        if err := rows.Scan(
            &m.ID, &m.UserID, &m.RecipientUserID, &m.Content, &m.CreatedAt,
        ); err != nil {
            return nil, err
        }
        messages = append(messages, m)
    }
    return messages, nil
}

// GetConversationMessages возвращает личную переписку между двумя пользователями.
// otherUsername — это e-mail (или username), по которому ищется пользователь в таблице users.
func (s *Service) GetConversationMessages(currentUserID, otherUsername string, limit int) ([]models.Message, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Получаем id другого пользователя по email/username
    var otherUserID string
    err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1`, otherUsername).Scan(&otherUserID)
    if err != nil {
        return nil, err
    }

    query := `
        SELECT id, user_id, recipient_user_id, content, created_at
        FROM messages
        WHERE 
            (user_id = $1 AND recipient_user_id = $2)
            OR
            (user_id = $2 AND recipient_user_id = $1)
        ORDER BY created_at DESC
        LIMIT $3
    `
    rows, err := s.db.QueryContext(ctx, query, currentUserID, otherUserID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var messages []models.Message
    for rows.Next() {
        var m models.Message
        if err := rows.Scan(
            &m.ID, &m.UserID, &m.RecipientUserID, &m.Content, &m.CreatedAt,
        ); err != nil {
            return nil, err
        }
        messages = append(messages, m)
    }
    return messages, nil
}

// SaveMessage сохраняет сообщение: recipientUserID == nil — в общий чат, иначе — приватное сообщение
func (s *Service) SaveMessage(userID string, recipientUserID *string, content string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    query := `INSERT INTO messages (user_id, recipient_user_id, content) VALUES ($1, $2, $3)`
    _, err := s.db.ExecContext(ctx, query, userID, recipientUserID, content)
    return err
}

func (s *Service) SaveAttachment(messageID, userID, filePath, fileName, mimeType string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    query := `INSERT INTO attachments (message_id, user_id, file_path, file_name, mime_type) VALUES ($1, $2, $3, $4, $5)`
    _, err := s.db.ExecContext(ctx, query, messageID, userID, filePath, fileName, mimeType)
    return err
}

func (s *Service) SaveMessageWithID(messageID, userID string, recipientUserID *string, content string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    query := `INSERT INTO messages (id, user_id, recipient_user_id, content) VALUES ($1, $2, $3, $4)`
    _, err := s.db.ExecContext(ctx, query, messageID, userID, recipientUserID, content)
    return err
}
