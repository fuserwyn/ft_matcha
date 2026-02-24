package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ActorID   *uuid.UUID
	Type      string
	EntityID  *uuid.UUID
	Content   string
	IsRead    bool
	CreatedAt time.Time
	ReadAt    *time.Time
}

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) Create(
	ctx context.Context,
	userID uuid.UUID,
	actorID *uuid.UUID,
	typ string,
	entityID *uuid.UUID,
	content string,
) (*Notification, error) {
	var n Notification
	err := r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, actor_id, type, entity_id, content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, actor_id, type, entity_id, content, is_read, created_at, read_at
	`, userID, actorID, typ, entityID, content).Scan(
		&n.ID, &n.UserID, &n.ActorID, &n.Type, &n.EntityID, &n.Content, &n.IsRead, &n.CreatedAt, &n.ReadAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NotificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]Notification, error) {
	query := `
		SELECT id, user_id, actor_id, type, entity_id, content, is_read, created_at, read_at
		FROM notifications
		WHERE user_id = $1
	`
	args := []any{userID}
	if unreadOnly {
		query += ` AND is_read = FALSE`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorID, &n.Type, &n.EntityID, &n.Content, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	res, err := r.pool.Exec(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}
