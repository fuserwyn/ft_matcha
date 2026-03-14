package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BlockRepository struct {
	pool *pgxpool.Pool
}

type BlockedUserCursorRow struct {
	BlockedUserID uuid.UUID
	CursorTime    time.Time
	CursorID      uuid.UUID
}

func NewBlockRepository(pool *pgxpool.Pool) *BlockRepository {
	return &BlockRepository{pool: pool}
}

func (r *BlockRepository) Block(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_blocks (blocker_user_id, blocked_user_id)
		VALUES ($1, $2)
		ON CONFLICT (blocker_user_id, blocked_user_id) DO NOTHING
	`, blockerID, blockedID)
	return err
}

func (r *BlockRepository) Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM user_blocks
		WHERE blocker_user_id = $1 AND blocked_user_id = $2
	`, blockerID, blockedID)
	return err
}

func (r *BlockRepository) IsBlockedEither(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	var blocked bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM user_blocks
			WHERE (blocker_user_id = $1 AND blocked_user_id = $2)
			   OR (blocker_user_id = $2 AND blocked_user_id = $1)
		)
	`, userA, userB).Scan(&blocked)
	return blocked, err
}

func (r *BlockRepository) BlockedBy(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	var blocked bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_blocks
			WHERE blocker_user_id = $1 AND blocked_user_id = $2
		)
	`, blockerID, blockedID).Scan(&blocked)
	return blocked, err
}

func (r *BlockRepository) ListBlockedIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT blocked_user_id
		FROM user_blocks
		WHERE blocker_user_id = $1
		UNION
		SELECT blocker_user_id
		FROM user_blocks
		WHERE blocked_user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *BlockRepository) ListBlockedByMe(ctx context.Context, userID uuid.UUID, limit, offset int) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT blocked_user_id
		FROM user_blocks
		WHERE blocker_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *BlockRepository) ListBlockedByMeCursor(ctx context.Context, userID uuid.UUID, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]BlockedUserCursorRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT blocked_user_id, created_at AS page_created_at, blocked_user_id AS page_id
		FROM user_blocks
		WHERE blocker_user_id = $1
		  AND (
		    $3::timestamptz IS NULL
		    OR created_at < $3
		    OR (created_at = $3 AND blocked_user_id < $4)
		  )
		ORDER BY created_at DESC, blocked_user_id DESC
		LIMIT $2
	`, userID, limit, cursorTime, cursorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []BlockedUserCursorRow
	for rows.Next() {
		var row BlockedUserCursorRow
		if err := rows.Scan(&row.BlockedUserID, &row.CursorTime, &row.CursorID); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
