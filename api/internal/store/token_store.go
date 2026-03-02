package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	prefixEmailVerify = "matcha:email_verify:"
	prefixPwdReset    = "matcha:pwd_reset:"
)

type TokenStore struct {
	client *redis.Client
}

func NewTokenStore(redisURL string) (*TokenStore, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis url: %w", err)
	}
	client := redis.NewClient(opt)
	return &TokenStore{client: client}, nil
}

func (s *TokenStore) Close() error {
	return s.client.Close()
}

// SetEmailVerify stores token -> userID, TTL 24h
func (s *TokenStore) SetEmailVerify(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	key := prefixEmailVerify + token
	return s.client.Set(ctx, key, userID.String(), ttl).Err()
}

// GetAndDeleteEmailVerify returns userID if token exists, then deletes it (one-time use)
func (s *TokenStore) GetAndDeleteEmailVerify(ctx context.Context, token string) (uuid.UUID, error) {
	key := prefixEmailVerify + token
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return uuid.Nil, ErrTokenNotFound
		}
		return uuid.Nil, err
	}
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// SetPwdReset stores tokenHash -> userID, TTL 30min
func (s *TokenStore) SetPwdReset(ctx context.Context, tokenHash string, userID uuid.UUID, ttl time.Duration) error {
	key := prefixPwdReset + tokenHash
	return s.client.Set(ctx, key, userID.String(), ttl).Err()
}

// GetAndDeletePwdReset returns userID if token exists, then deletes it (one-time use)
func (s *TokenStore) GetAndDeletePwdReset(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	key := prefixPwdReset + tokenHash
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return uuid.Nil, ErrTokenNotFound
		}
		return uuid.Nil, err
	}
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
