CREATE TABLE IF NOT EXISTS message_requests (
    id              TEXT PRIMARY KEY,
    sender_id       TEXT NOT NULL REFERENCES users(id),
    recipient_id    TEXT NOT NULL REFERENCES users(id),
    initial_message TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','accepted','declined','blocked')),
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    UNIQUE(sender_id, recipient_id)
);
CREATE INDEX IF NOT EXISTS idx_requests_recipient ON message_requests(recipient_id, status);
