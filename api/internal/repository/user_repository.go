package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID              uuid.UUID
	Username        string
	Email           string
	PasswordHash    string
	FirstName       string
	LastName        string
	EmailVerifiedAt sql.NullTime
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	return total, err
}

func (r *UserRepository) Create(ctx context.Context, u *User) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, first_name, last_name, email_verified_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, u.ID, u.Username, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.EmailVerifiedAt)
	return err
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, first_name, last_name, email_verified_at
		FROM users WHERE username = $1
	`, username).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.EmailVerifiedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, first_name, last_name, email_verified_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.EmailVerifiedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, first_name, last_name, email_verified_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.EmailVerifiedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) ListIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM users`)
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
	return ids, nil
}

func (r *UserRepository) SetEmailVerified(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET email_verified_at = COALESCE(email_verified_at, NOW())
		WHERE id = $1
	`, userID)
	return err
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2
		WHERE id = $1
	`, userID, passwordHash)
	return err
}

func (r *UserRepository) StorePasswordResetToken(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, tokenHash, userID, expiresAt)
	return err
}

func (r *UserRepository) GetUserIDByValidResetToken(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT user_id
		FROM password_reset_tokens
		WHERE token_hash = $1
			AND used_at IS NULL
			AND expires_at > NOW()
	`, tokenHash).Scan(&userID)
	return userID, err
}

func (r *UserRepository) MarkPasswordResetTokenUsed(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE token_hash = $1
			AND used_at IS NULL
	`, tokenHash)
	return err
}

func (r *UserRepository) UpdateAccount(ctx context.Context, userID uuid.UUID, username, email, firstName, lastName string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET username = $2,
		    email = $3,
		    first_name = $4,
		    last_name = $5
		WHERE id = $1
	`, userID, username, email, firstName, lastName)
	return err
}
