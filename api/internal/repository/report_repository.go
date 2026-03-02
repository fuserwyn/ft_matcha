package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserReport struct {
	ID             uuid.UUID
	ReporterUserID uuid.UUID
	TargetUserID   uuid.UUID
	Reason         string
	Comment        *string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ReportRepository struct {
	pool *pgxpool.Pool
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

func (r *ReportRepository) Upsert(ctx context.Context, reporterUserID, targetUserID uuid.UUID, reason string, comment *string) (*UserReport, error) {
	var report UserReport
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_reports (reporter_user_id, target_user_id, reason, comment)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_user_id, target_user_id)
		DO UPDATE SET
			reason = EXCLUDED.reason,
			comment = EXCLUDED.comment,
			status = 'open',
			updated_at = NOW()
		RETURNING id, reporter_user_id, target_user_id, reason, comment, status, created_at, updated_at
	`, reporterUserID, targetUserID, reason, comment).Scan(
		&report.ID,
		&report.ReporterUserID,
		&report.TargetUserID,
		&report.Reason,
		&report.Comment,
		&report.Status,
		&report.CreatedAt,
		&report.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *ReportRepository) ListByReporter(ctx context.Context, reporterUserID uuid.UUID, limit, offset int) ([]UserReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, reporter_user_id, target_user_id, reason, comment, status, created_at, updated_at
		FROM user_reports
		WHERE reporter_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, reporterUserID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []UserReport
	for rows.Next() {
		var item UserReport
		if err := rows.Scan(
			&item.ID,
			&item.ReporterUserID,
			&item.TargetUserID,
			&item.Reason,
			&item.Comment,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		reports = append(reports, item)
	}
	return reports, rows.Err()
}
