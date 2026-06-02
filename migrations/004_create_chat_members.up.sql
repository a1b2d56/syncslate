CREATE TABLE IF NOT EXISTS chat_members (
    chat_id    TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'member' CHECK(role IN ('owner','admin','member')),
    joined_at  INTEGER NOT NULL,
    PRIMARY KEY (chat_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_chat_members_user ON chat_members(user_id);
