package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID           uuid.UUID
	Username     string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func (r *UserRepository) Create(ctx context.Context, u *User) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, first_name, last_name)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, u.ID, u.Username, u.Email, u.PasswordHash, u.FirstName, u.LastName)
	return err
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, first_name, last_name
		FROM users WHERE username = $1
	`, username).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, first_name, last_name
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
