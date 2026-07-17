CREATE TABLE IF NOT EXISTS media (
    id          TEXT PRIMARY KEY,
    uploader_id TEXT NOT NULL REFERENCES users(id),
    file_name   TEXT NOT NULL,
    file_type   TEXT NOT NULL,
    file_size   INTEGER NOT NULL,
    file_path   TEXT NOT NULL,
    created_at  INTEGER NOT NULL
);
