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

func (s *Service) SaveMessage(userID, content string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    query := `INSERT INTO messages (user_id, content) VALUES ($1, $2)`
    _, err := s.db.ExecContext(ctx, query, userID, content)
    return err
}

func (s *Service) GetMessages(limit int) ([]models.Message, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    query := `SELECT id, user_id, content, created_at FROM messages ORDER BY created_at DESC LIMIT $1`
    rows, err := s.db.QueryContext(ctx, query, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var messages []models.Message
    for rows.Next() {
        var m models.Message
        if err := rows.Scan(&m.ID, &m.UserID, &m.Content, &m.CreatedAt); err != nil {
            return nil, err
        }
        messages = append(messages, m)
    }
    return messages, nil
}
