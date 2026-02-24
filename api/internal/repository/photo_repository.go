package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Photo struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ObjectKey string
	URL       string
	IsPrimary bool
	Position  int
	CreatedAt time.Time
}

type PhotoRepository struct {
	pool *pgxpool.Pool
}

func NewPhotoRepository(pool *pgxpool.Pool) *PhotoRepository {
	return &PhotoRepository{pool: pool}
}

func (r *PhotoRepository) Create(ctx context.Context, userID uuid.UUID, objectKey, url string, makePrimary bool) (*Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if makePrimary {
		if _, err := tx.Exec(ctx, `UPDATE user_photos SET is_primary = FALSE WHERE user_id = $1`, userID); err != nil {
			return nil, err
		}
	}

	var p Photo
	err = tx.QueryRow(ctx, `
		WITH next_pos AS (
			SELECT COALESCE(MAX(position), 0) + 1 AS pos FROM user_photos WHERE user_id = $1
		)
		INSERT INTO user_photos (user_id, object_key, url, is_primary, position)
		VALUES ($1, $2, $3, $4, (SELECT pos FROM next_pos))
		RETURNING id, user_id, object_key, url, is_primary, position, created_at
	`, userID, objectKey, url, makePrimary).Scan(
		&p.ID, &p.UserID, &p.ObjectKey, &p.URL, &p.IsPrimary, &p.Position, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PhotoRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Photo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, object_key, url, is_primary, position, created_at
		FROM user_photos
		WHERE user_id = $1
		ORDER BY is_primary DESC, position ASC, created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Photo
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.UserID, &p.ObjectKey, &p.URL, &p.IsPrimary, &p.Position, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PhotoRepository) GetPrimaryByUser(ctx context.Context, userID uuid.UUID) (*Photo, error) {
	var p Photo
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, object_key, url, is_primary, position, created_at
		FROM user_photos
		WHERE user_id = $1
		ORDER BY is_primary DESC, position ASC, created_at ASC
		LIMIT 1
	`, userID).Scan(&p.ID, &p.UserID, &p.ObjectKey, &p.URL, &p.IsPrimary, &p.Position, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *PhotoRepository) SetPrimary(ctx context.Context, userID, photoID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `UPDATE user_photos SET is_primary = FALSE WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE user_photos SET is_primary = TRUE WHERE user_id = $1 AND id = $2`, userID, photoID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PhotoRepository) GetByID(ctx context.Context, photoID uuid.UUID) (*Photo, error) {
	var p Photo
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, object_key, url, is_primary, position, created_at
		FROM user_photos
		WHERE id = $1
	`, photoID).Scan(&p.ID, &p.UserID, &p.ObjectKey, &p.URL, &p.IsPrimary, &p.Position, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *PhotoRepository) DeleteByID(ctx context.Context, userID, photoID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM user_photos WHERE id = $1 AND user_id = $2`, photoID, userID)
	return err
}
