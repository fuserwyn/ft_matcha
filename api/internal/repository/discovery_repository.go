package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"matcha/api/internal/search"
)

type UserCard struct {
	ID           uuid.UUID
	Username     string
	FirstName    string
	LastName     string
	Gender       *string
	BirthDate    *time.Time
	Bio          *string
	FameRating   int
	Latitude     *float64
	Longitude    *float64
}

type DiscoveryFilters struct {
	ExcludeID   uuid.UUID
	Gender      string
	Interest    string
	MinAge      int
	MaxAge      int
	Limit       int
	Offset      int
}

func NewDiscoveryRepository(searchClient *search.Client) *DiscoveryRepository {
	return &DiscoveryRepository{search: searchClient}
}

type DiscoveryRepository struct {
	search *search.Client
}

func (r *DiscoveryRepository) Search(ctx context.Context, f DiscoveryFilters) ([]UserCard, error) {
	sf := search.SearchFilters{
		ExcludeID: f.ExcludeID,
		Gender:    f.Gender,
		Interest:  f.Interest,
		MinAge:    f.MinAge,
		MaxAge:    f.MaxAge,
		Limit:     f.Limit,
		Offset:    f.Offset,
	}
	docs, err := r.search.Search(ctx, sf)
	if err != nil {
		return nil, err
	}
	cards := make([]UserCard, len(docs))
	for i, d := range docs {
		id, _ := uuid.Parse(d.UserID)
		cards[i] = UserCard{
			ID:        id,
			Username:  d.Username,
			FirstName: d.FirstName,
			LastName:  d.LastName,
			Bio:       strPtr(d.Bio),
			FameRating: d.FameRating,
		}
		if d.Gender != "" {
			cards[i].Gender = &d.Gender
		}
		if d.BirthDate != "" {
			t, _ := time.Parse("2006-01-02", d.BirthDate)
			cards[i].BirthDate = &t
		}
		if d.Location != nil {
			lat, lon := d.Location.Lat, d.Location.Lon
			cards[i].Latitude = &lat
			cards[i].Longitude = &lon
		}
	}
	return cards, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
