package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewLikeRepository(pool *pgxpool.Pool) *LikeRepository {
	return &LikeRepository{pool: pool}
}

type LikeRepository struct {
	pool *pgxpool.Pool
}

func (r *LikeRepository) Create(ctx context.Context, userID, likedUserID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO likes (user_id, liked_user_id)
		VALUES ($1, $2)
	`, userID, likedUserID)
	return err
}

func (r *LikeRepository) Delete(ctx context.Context, userID, likedUserID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM likes WHERE user_id = $1 AND liked_user_id = $2
	`, userID, likedUserID)
	return err
}

func (r *LikeRepository) Exists(ctx context.Context, userID, likedUserID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = $1 AND liked_user_id = $2)
	`, userID, likedUserID).Scan(&exists)
	return exists, err
}

func (r *LikeRepository) IsMatch(ctx context.Context, userID, otherUserID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM likes l1
			JOIN likes l2 ON l1.user_id = l2.liked_user_id AND l1.liked_user_id = l2.user_id
			WHERE l1.user_id = $1 AND l1.liked_user_id = $2
		)
	`, userID, otherUserID).Scan(&exists)
	return exists, err
}

func (r *LikeRepository) GetLikedByMe(ctx context.Context, userID uuid.UUID, limit, offset int) ([]UserCard, error) {
	return r.getUserCardsFromLikes(ctx, `
		SELECT u.id, u.username, u.first_name, u.last_name,
		       p.gender, p.birth_date, p.bio, p.fame_rating, p.latitude, p.longitude
		FROM likes l
		JOIN users u ON u.id = l.liked_user_id
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE l.user_id = $1
		ORDER BY l.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
}

func (r *LikeRepository) GetLikedMe(ctx context.Context, userID uuid.UUID, limit, offset int) ([]UserCard, error) {
	return r.getUserCardsFromLikes(ctx, `
		SELECT u.id, u.username, u.first_name, u.last_name,
		       p.gender, p.birth_date, p.bio, p.fame_rating, p.latitude, p.longitude
		FROM likes l
		JOIN users u ON u.id = l.user_id
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE l.liked_user_id = $1
		ORDER BY l.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
}

func (r *LikeRepository) GetMatches(ctx context.Context, userID uuid.UUID, limit, offset int) ([]UserCard, error) {
	return r.getUserCardsFromLikes(ctx, `
		SELECT u.id, u.username, u.first_name, u.last_name,
		       p.gender, p.birth_date, p.bio, p.fame_rating, p.latitude, p.longitude
		FROM likes l1
		JOIN likes l2 ON l1.user_id = l2.liked_user_id AND l1.liked_user_id = l2.user_id
		JOIN users u ON u.id = l1.liked_user_id
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE l1.user_id = $1
		ORDER BY l1.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
}

func (r *LikeRepository) getUserCardsFromLikes(ctx context.Context, query string, userID uuid.UUID, limit, offset int) ([]UserCard, error) {
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []UserCard
	for rows.Next() {
		var c UserCard
		var gender, bio sql.NullString
		var birthDate sql.NullTime
		var fameRating sql.NullInt64
		var lat, lon sql.NullFloat64

		err := rows.Scan(
			&c.ID, &c.Username, &c.FirstName, &c.LastName,
			&gender, &birthDate, &bio, &fameRating, &lat, &lon,
		)
		if err != nil {
			return nil, err
		}
		if gender.Valid {
			c.Gender = &gender.String
		}
		if birthDate.Valid {
			c.BirthDate = &birthDate.Time
		}
		if bio.Valid {
			c.Bio = &bio.String
		}
		if fameRating.Valid {
			c.FameRating = int(fameRating.Int64)
		}
		if lat.Valid {
			c.Latitude = &lat.Float64
		}
		if lon.Valid {
			c.Longitude = &lon.Float64
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}
