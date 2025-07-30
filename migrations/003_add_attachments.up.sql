CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS attachments
(
    id         SERIAL PRIMARY KEY,
    message_id INTEGER REFERENCES messages (id) ON DELETE CASCADE,
    user_id    INTEGER REFERENCES users (id) ON DELETE CASCADE,
    file_path  VARCHAR(255) NOT NULL,
    file_name  VARCHAR(255) NOT NULL,
    mime_type  VARCHAR(100) NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Индекс для ускорения поиска по сообщению
CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments (message_id);

-- Индекс для ускорения поиска по пользователю
CREATE INDEX IF NOT EXISTS idx_attachments_user_id ON attachments (user_id);