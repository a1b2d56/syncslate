DROP TABLE IF EXISTS chat_folder_chats;
DROP TABLE IF EXISTS chat_folders;
DROP TABLE IF EXISTS chat_pins;
DROP TABLE IF EXISTS chat_mutes;
-- SQLite does not easily support ALTER TABLE DROP COLUMN in older syntax without rebuilding, but table drop is standard for down migration
