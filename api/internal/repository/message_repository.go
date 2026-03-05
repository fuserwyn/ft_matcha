package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

type MessageRepository struct {
	pool *pgxpool.Pool
}

type Message struct {
	ID          uuid.UUID
	SenderID    uuid.UUID
	ReceiverID  uuid.UUID
	Content     string
	MessageType string
	MediaURL    *string
	CreatedAt   time.Time
	IsRead      bool
	ReadAt      *time.Time
}

func (r *MessageRepository) Create(ctx context.Context, senderID, receiverID uuid.UUID, content string) (*Message, error) {
	return r.CreateWithMeta(ctx, senderID, receiverID, content, "text", nil)
}

func (r *MessageRepository) CreateWithMeta(ctx context.Context, senderID, receiverID uuid.UUID, content string, messageType string, mediaURL *string) (*Message, error) {
	var m Message
	err := r.pool.QueryRow(ctx, `
		INSERT INTO messages (sender_id, receiver_id, content, message_type, media_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, sender_id, receiver_id, content, message_type, media_url, created_at, is_read, read_at
	`, senderID, receiverID, content, messageType, mediaURL).Scan(
		&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.MessageType, &m.MediaURL, &m.CreatedAt, &m.IsRead, &m.ReadAt,
	)
	return &m, err
}

func (r *MessageRepository) GetBetween(ctx context.Context, userID, otherUserID uuid.UUID, limit, offset int) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sender_id, receiver_id, content, message_type, media_url, created_at, is_read, read_at
		FROM messages
		WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)
		ORDER BY created_at ASC
		LIMIT $3 OFFSET $4
	`, userID, otherUserID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var readAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.MessageType, &m.MediaURL, &m.CreatedAt, &m.IsRead, &readAt); err != nil {
			return nil, err
		}
		if readAt.Valid {
			t := readAt.Time
			m.ReadAt = &t
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *MessageRepository) MarkReadFromSender(ctx context.Context, receiverID, senderID uuid.UUID) (int64, error) {
	res, err := r.pool.Exec(ctx, `
		UPDATE messages
		SET is_read = TRUE, read_at = NOW()
		WHERE receiver_id = $1 AND sender_id = $2 AND is_read = FALSE
	`, receiverID, senderID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}
