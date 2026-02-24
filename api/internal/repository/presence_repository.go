package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PresenceRepository struct {
	pool *pgxpool.Pool
}

func NewPresenceRepository(pool *pgxpool.Pool) *PresenceRepository {
	return &PresenceRepository{pool: pool}
}

func (r *PresenceRepository) UpsertLastSeen(ctx context.Context, userID uuid.UUID, at time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_presence (user_id, last_seen, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET last_seen = EXCLUDED.last_seen, updated_at = NOW()
	`, userID, at)
	return err
}

func (r *PresenceRepository) GetLastSeen(ctx context.Context, userID uuid.UUID) (*time.Time, error) {
	var lastSeen time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT last_seen FROM user_presence WHERE user_id = $1
	`, userID).Scan(&lastSeen)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &lastSeen, nil
}
