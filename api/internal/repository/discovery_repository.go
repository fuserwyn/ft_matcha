package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"matcha/api/internal/search"
)

type UserCard struct {
	ID         uuid.UUID
	Username   string
	FirstName  string
	LastName   string
	Gender     *string
	BirthDate  *time.Time
	Bio        *string
	City       *string
	Tags       []string
	FameRating int
	Latitude   *float64
	Longitude  *float64
}

type DiscoveryFilters struct {
	ExcludeID     uuid.UUID
	ExcludeIDs    []uuid.UUID
	Genders       []string
	Interests     []string
	Tags          []string
	StrictTags    bool
	City          string
	PreferredCity string
	MinAge        int
	MaxAge        int
	MinFame       int
	MaxFame       int
	UserLat       *float64
	UserLon       *float64
	MaxDistanceKm int
	SortBy        string
	SortOrder     string
	Limit         int
	Offset        int
}

func NewDiscoveryRepository(searchClient *search.Client) *DiscoveryRepository {
	return &DiscoveryRepository{search: searchClient}
}

type DiscoveryRepository struct {
	search *search.Client
}

func (r *DiscoveryRepository) SearchCities(ctx context.Context, prefix string, limit int) ([]string, error) {
	return r.search.SearchCities(ctx, prefix, limit)
}

func (r *DiscoveryRepository) SearchTags(ctx context.Context, prefix string, limit int) ([]string, error) {
	return r.search.SearchTags(ctx, prefix, limit)
}

func (r *DiscoveryRepository) FilterAggregations(ctx context.Context, excludeID uuid.UUID, excludeIDs []uuid.UUID) (gender map[string]int64, interest map[string]int64, err error) {
	return r.search.FilterAggregations(ctx, excludeID, excludeIDs)
}

func (r *DiscoveryRepository) Search(ctx context.Context, f DiscoveryFilters) ([]UserCard, error) {
	sf := search.SearchFilters{
		ExcludeID:     f.ExcludeID,
		ExcludeIDs:    f.ExcludeIDs,
		Genders:       f.Genders,
		Interests:     f.Interests,
		Tags:          f.Tags,
		StrictTags:    f.StrictTags,
		City:          f.City,
		PreferredCity: f.PreferredCity,
		MinAge:        f.MinAge,
		MaxAge:        f.MaxAge,
		MinFame:       f.MinFame,
		MaxFame:       f.MaxFame,
		UserLat:       f.UserLat,
		UserLon:       f.UserLon,
		MaxDistanceKm: f.MaxDistanceKm,
		SortBy:        f.SortBy,
		SortOrder:     f.SortOrder,
		Limit:         f.Limit,
		Offset:        f.Offset,
	}
	docs, err := r.search.Search(ctx, sf)
	if err != nil {
		return nil, err
	}
	cards := make([]UserCard, len(docs))
	for i, d := range docs {
		id, _ := uuid.Parse(d.UserID)
		cards[i] = UserCard{
			ID:         id,
			Username:   d.Username,
			FirstName:  d.FirstName,
			LastName:   d.LastName,
			Bio:        strPtr(d.Bio),
			City:       strPtr(d.City),
			Tags:       d.Tags,
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
