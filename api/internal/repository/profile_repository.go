package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Profile struct {
	UserID           uuid.UUID
	Bio              *string
	Gender           *string
	SexualPreference *string
	BirthDate        *time.Time
	Latitude         *float64
	Longitude        *float64
	FameRating       int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewProfileRepository(pool *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{pool: pool}
}

type ProfileRepository struct {
	pool *pgxpool.Pool
}

func (r *ProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var p Profile
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, bio, gender, sexual_preference, birth_date,
		       latitude, longitude, fame_rating, created_at, updated_at
		FROM profiles WHERE user_id = $1
	`, userID).Scan(
		&p.UserID,
		&p.Bio,
		&p.Gender,
		&p.SexualPreference,
		&p.BirthDate,
		&p.Latitude,
		&p.Longitude,
		&p.FameRating,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepository) Upsert(ctx context.Context, p *Profile) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO profiles (user_id, bio, gender, sexual_preference, birth_date,
		                     latitude, longitude, fame_rating, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			bio = COALESCE(EXCLUDED.bio, profiles.bio),
			gender = COALESCE(EXCLUDED.gender, profiles.gender),
			sexual_preference = COALESCE(EXCLUDED.sexual_preference, profiles.sexual_preference),
			birth_date = COALESCE(EXCLUDED.birth_date, profiles.birth_date),
			latitude = COALESCE(EXCLUDED.latitude, profiles.latitude),
			longitude = COALESCE(EXCLUDED.longitude, profiles.longitude),
			updated_at = NOW()
	`, p.UserID, p.Bio, p.Gender, p.SexualPreference, p.BirthDate,
		p.Latitude, p.Longitude, p.FameRating)
	return err
}
