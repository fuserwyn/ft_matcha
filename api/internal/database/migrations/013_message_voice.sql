ALTER TABLE messages
ADD COLUMN IF NOT EXISTS message_type TEXT NOT NULL DEFAULT 'text',
ADD COLUMN IF NOT EXISTS media_url TEXT;

CREATE INDEX IF NOT EXISTS idx_messages_type_created_at
ON messages(message_type, created_at DESC);
