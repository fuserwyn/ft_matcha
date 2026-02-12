package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"matcha/api/internal/repository"
)

var ErrInvalidPassword = errors.New("invalid password")
var ErrUserExists = errors.New("user exists")

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

type AuthService struct {
	userRepo *repository.UserRepository
}

func (s *AuthService) Register(ctx context.Context, username, email, password, firstName, lastName string) (*repository.User, error) {
	if len(password) < 8 {
		return nil, errors.New("password min 8 chars")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	_, err = s.userRepo.GetByUsername(ctx, username)
	if err == nil {
		return nil, ErrUserExists
	}
	_, err = s.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserExists
	}

	u := &repository.User{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    firstName,
		LastName:     lastName,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *AuthService) GetByID(ctx context.Context, id uuid.UUID) (*repository.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*repository.User, error) {
	u, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidPassword
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidPassword
	}
	return u, nil
}
