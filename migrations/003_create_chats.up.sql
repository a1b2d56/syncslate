CREATE TABLE IF NOT EXISTS chats (
    id         TEXT PRIMARY KEY,
    type       TEXT NOT NULL DEFAULT 'direct' CHECK(type IN ('direct','group','channel')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
