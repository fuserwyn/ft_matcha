package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"matcha/api/internal/repository"
	"matcha/api/internal/search"
)

type SyncService struct {
	userRepo    *repository.UserRepository
	profileRepo *repository.ProfileRepository
	search      *search.Client
}

func NewSyncService(userRepo *repository.UserRepository, profileRepo *repository.ProfileRepository, searchClient *search.Client) *SyncService {
	return &SyncService{
		userRepo:    userRepo,
		profileRepo: profileRepo,
		search:      searchClient,
	}
}

func (s *SyncService) SyncUser(ctx context.Context, userID uuid.UUID) error {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	doc := &search.UserDoc{
		UserID:    u.ID.String(),
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	p, err := s.profileRepo.GetByUserID(ctx, userID)
	if err == nil && p != nil {
		if p.Gender != nil {
			doc.Gender = *p.Gender
		}
		if p.SexualPreference != nil {
			doc.SexualPreference = *p.SexualPreference
		}
		if p.BirthDate != nil {
			doc.BirthDate = p.BirthDate.Format("2006-01-02")
		}
		if p.Bio != nil {
			doc.Bio = *p.Bio
		}
		doc.FameRating = p.FameRating
		if p.Latitude != nil && p.Longitude != nil {
			doc.Location = &search.GeoPoint{Lat: *p.Latitude, Lon: *p.Longitude}
		}
	}
	return s.search.Index(ctx, doc)
}

func (s *SyncService) RemoveUser(ctx context.Context, userID uuid.UUID) error {
	return s.search.Delete(ctx, userID.String())
}

// ReindexAll syncs all users from PostgreSQL to Elasticsearch
func (s *SyncService) ReindexAll(ctx context.Context) error {
	ids, err := s.userRepo.ListIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.SyncUser(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
