CREATE TABLE IF NOT EXISTS read_receipts (
    chat_id             TEXT NOT NULL REFERENCES chats(id),
    user_id             TEXT NOT NULL REFERENCES users(id),
    last_read_message_id TEXT NOT NULL,
    updated_at          INTEGER NOT NULL,
    PRIMARY KEY (chat_id, user_id)
);
