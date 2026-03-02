package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"matcha/api/internal/repository"
)

var ErrInvalidPassword = errors.New("invalid password")
var ErrUserExists = errors.New("user exists")
var ErrInvalidResetToken = errors.New("invalid reset token")

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

type AuthService struct {
	userRepo *repository.UserRepository
}

const passwordResetTTL = 30 * time.Minute

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

func (s *AuthService) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	return s.userRepo.SetEmailVerified(ctx, userID)
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (string, *repository.User, error) {
	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, nil
	}

	resetToken, tokenHash, err := generateResetToken()
	if err != nil {
		return "", nil, err
	}
	if err := s.userRepo.StorePasswordResetToken(ctx, tokenHash, u.ID, time.Now().Add(passwordResetTTL)); err != nil {
		return "", nil, err
	}
	return resetToken, u, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password min 8 chars")
	}

	tokenHash := hashToken(resetToken)
	userID, err := s.userRepo.GetUserIDByValidResetToken(ctx, tokenHash)
	if err != nil {
		return ErrInvalidResetToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.userRepo.UpdatePasswordHash(ctx, userID, string(hash)); err != nil {
		return err
	}
	if err := s.userRepo.MarkPasswordResetTokenUsed(ctx, tokenHash); err != nil {
		return err
	}
	return nil
}

func generateResetToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
