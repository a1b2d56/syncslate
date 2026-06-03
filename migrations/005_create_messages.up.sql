CREATE TABLE IF NOT EXISTS messages (
    id            TEXT PRIMARY KEY,
    chat_id       TEXT NOT NULL REFERENCES chats(id),
    sender_id     TEXT NOT NULL REFERENCES users(id),
    content       TEXT NOT NULL DEFAULT '',
    media_id      TEXT,
    reply_to_id   TEXT,
    is_edited     INTEGER NOT NULL DEFAULT 0,
    edited_at     INTEGER,
    is_deleted    INTEGER NOT NULL DEFAULT 0,
    deleted_at    INTEGER,
    created_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_chat_time ON messages(chat_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);
