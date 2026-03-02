package repository

import (
	"context"
	"database/sql"
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
	City             *string
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

func (r *ProfileRepository) CountByGender(ctx context.Context) (male int, female int, err error) {
	rows, err := r.pool.Query(ctx, `
		SELECT gender, COUNT(*)::int
		FROM profiles
		WHERE gender IN ('male', 'female')
		GROUP BY gender
	`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var gender string
		var count int
		if err := rows.Scan(&gender, &count); err != nil {
			return 0, 0, err
		}
		switch gender {
		case "male":
			male = count
		case "female":
			female = count
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	return male, female, nil
}

func (r *ProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var p Profile
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, bio, gender, sexual_preference, birth_date,
		       city, latitude, longitude, fame_rating, created_at, updated_at
		FROM profiles WHERE user_id = $1
	`, userID).Scan(
		&p.UserID,
		&p.Bio,
		&p.Gender,
		&p.SexualPreference,
		&p.BirthDate,
		&p.City,
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
		                     city, latitude, longitude, fame_rating, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			bio = COALESCE(EXCLUDED.bio, profiles.bio),
			gender = COALESCE(EXCLUDED.gender, profiles.gender),
			sexual_preference = COALESCE(EXCLUDED.sexual_preference, profiles.sexual_preference),
			birth_date = COALESCE(EXCLUDED.birth_date, profiles.birth_date),
			city = COALESCE(EXCLUDED.city, profiles.city),
			latitude = COALESCE(EXCLUDED.latitude, profiles.latitude),
			longitude = COALESCE(EXCLUDED.longitude, profiles.longitude),
			updated_at = NOW()
	`, p.UserID, p.Bio, p.Gender, p.SexualPreference, p.BirthDate,
		p.City, p.Latitude, p.Longitude, p.FameRating)
	return err
}

func (r *ProfileRepository) SetTags(ctx context.Context, userID uuid.UUID, tags []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM user_tags WHERE user_id = $1`, userID); err != nil {
		return err
	}

	for _, tag := range tags {
		var tagID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO tags (name)
			VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, tag).Scan(&tagID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_tags (user_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT (user_id, tag_id) DO NOTHING
		`, userID, tagID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ProfileRepository) GetTags(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.name
		FROM user_tags ut
		JOIN tags t ON t.id = ut.tag_id
		WHERE ut.user_id = $1
		ORDER BY t.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *ProfileRepository) ListTopTags(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.pool.Query(ctx, `
		SELECT t.name
		FROM user_tags ut
		JOIN tags t ON t.id = ut.tag_id
		GROUP BY t.name
		ORDER BY COUNT(*) DESC, t.name ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *ProfileRepository) AddProfileView(ctx context.Context, viewerUserID, viewedUserID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO profile_views (viewer_user_id, viewed_user_id)
		VALUES ($1, $2)
	`, viewerUserID, viewedUserID)
	return err
}

type ViewedProfile struct {
	UserID       uuid.UUID
	Username     string
	FirstName    string
	LastName     string
	City         *string
	FameRating   int
	LastViewedAt time.Time
}

func (r *ProfileRepository) GetViewedProfiles(ctx context.Context, viewerUserID uuid.UUID, limit, offset int) ([]ViewedProfile, error) {
	rows, err := r.pool.Query(ctx, `
		WITH latest AS (
			SELECT viewed_user_id, MAX(created_at) AS last_viewed_at
			FROM profile_views
			WHERE viewer_user_id = $1
			GROUP BY viewed_user_id
		)
		SELECT
			u.id, u.username, u.first_name, u.last_name,
			p.city, COALESCE(p.fame_rating, 0), l.last_viewed_at
		FROM latest l
		JOIN users u ON u.id = l.viewed_user_id
		LEFT JOIN profiles p ON p.user_id = u.id
		ORDER BY l.last_viewed_at DESC
		LIMIT $2 OFFSET $3
	`, viewerUserID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ViewedProfile
	for rows.Next() {
		var item ViewedProfile
		var city sql.NullString
		if err := rows.Scan(
			&item.UserID,
			&item.Username,
			&item.FirstName,
			&item.LastName,
			&city,
			&item.FameRating,
			&item.LastViewedAt,
		); err != nil {
			return nil, err
		}
		if city.Valid {
			item.City = &city.String
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *ProfileRepository) RecalculateFameRating(ctx context.Context, userID uuid.UUID) (int, error) {
	var score int
	err := r.pool.QueryRow(ctx, `
		WITH likes_count AS (
			SELECT COUNT(*)::int AS c
			FROM likes
			WHERE liked_user_id = $1
		),
		views_count AS (
			SELECT COUNT(*)::int AS c
			FROM profile_views
			WHERE viewed_user_id = $1
		)
		SELECT COALESCE(l.c, 0) * 5 + COALESCE(v.c, 0)
		FROM likes_count l, views_count v
	`, userID).Scan(&score)
	if err != nil {
		return 0, err
	}

	if _, err := r.pool.Exec(ctx, `
		INSERT INTO profiles (user_id, fame_rating, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET fame_rating = EXCLUDED.fame_rating,
		    updated_at = NOW()
	`, userID, score); err != nil {
		return 0, err
	}
	return score, nil
}
