ALTER TABLE messages
ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_messages_receiver_read
ON messages(receiver_id, is_read, created_at DESC);
