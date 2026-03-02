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
	"matcha/api/internal/validation"
)

var ErrInvalidPassword = errors.New("invalid password")
var ErrUserExists = errors.New("user exists")
var ErrInvalidResetToken = errors.New("invalid reset token")
var ErrEmailNotVerified = errors.New("email not verified")

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

type AuthService struct {
	userRepo *repository.UserRepository
}

const passwordResetTTL = 30 * time.Minute

func (s *AuthService) Register(ctx context.Context, username, email, password, firstName, lastName string) (*repository.User, error) {
	if err := validation.ValidatePassword(password); err != nil {
		return nil, err
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
	if !u.EmailVerifiedAt.Valid {
		return nil, ErrEmailNotVerified
	}
	return u, nil
}

func (s *AuthService) UpdateAccount(ctx context.Context, userID uuid.UUID, username, email, firstName, lastName string) error {
	if err := validation.ValidateUsername(username); err != nil {
		return err
	}
	if err := validation.ValidateEmail(email); err != nil {
		return err
	}
	if err := validation.ValidateName(firstName, "first_name"); err != nil {
		return err
	}
	if err := validation.ValidateName(lastName, "last_name"); err != nil {
		return err
	}

	if u, err := s.userRepo.GetByUsername(ctx, username); err == nil && u.ID != userID {
		return ErrUserExists
	}
	if u, err := s.userRepo.GetByEmail(ctx, email); err == nil && u.ID != userID {
		return ErrUserExists
	}
	return s.userRepo.UpdateAccount(ctx, userID, username, email, firstName, lastName)
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
	if err := validation.ValidatePassword(newPassword); err != nil {
		return err
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
