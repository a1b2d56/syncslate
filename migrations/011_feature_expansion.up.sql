-- Add last_seen to users
ALTER TABLE users ADD COLUMN last_seen INTEGER DEFAULT NULL;

-- Add pinned_message_id to chats
ALTER TABLE chats ADD COLUMN pinned_message_id TEXT REFERENCES messages(id) ON DELETE SET NULL;

-- Chat mutes table (per user per chat)
CREATE TABLE IF NOT EXISTS chat_mutes (
    chat_id    TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mute_until INTEGER, -- NULL means permanent mute
    created_at INTEGER NOT NULL,
    PRIMARY KEY (chat_id, user_id)
);

-- Pinned chats (per user per chat)
CREATE TABLE IF NOT EXISTS chat_pins (
    chat_id      TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pinned_order INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL,
    PRIMARY KEY (chat_id, user_id)
);

-- Chat folders table
CREATE TABLE IF NOT EXISTS chat_folders (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    icon         TEXT NOT NULL DEFAULT '',
    filter_flags INTEGER NOT NULL DEFAULT 0,
    folder_order INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chat_folders_user ON chat_folders(user_id);

-- Explicit chats added to folder
CREATE TABLE IF NOT EXISTS chat_folder_chats (
    folder_id TEXT NOT NULL REFERENCES chat_folders(id) ON DELETE CASCADE,
    chat_id   TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    PRIMARY KEY (folder_id, chat_id)
);
