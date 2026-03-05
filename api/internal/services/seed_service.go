package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"matcha/api/internal/repository"
)

type SeedService struct {
	userRepo    *repository.UserRepository
	profileRepo *repository.ProfileRepository
	photoRepo   *repository.PhotoRepository
}

func NewSeedService(
	userRepo *repository.UserRepository,
	profileRepo *repository.ProfileRepository,
	photoRepo *repository.PhotoRepository,
) *SeedService {
	return &SeedService{userRepo: userRepo, profileRepo: profileRepo, photoRepo: photoRepo}
}

// randomUserResult holds the fields we need from randomuser.me API.
type randomUserResult struct {
	Name struct {
		First string `json:"first"`
		Last  string `json:"last"`
	} `json:"name"`
	Dob struct {
		Date string `json:"date"`
		Age  int    `json:"age"`
	} `json:"dob"`
	Location struct {
		City        string `json:"city"`
		Coordinates struct {
			Latitude  string `json:"latitude"`
			Longitude string `json:"longitude"`
		} `json:"coordinates"`
	} `json:"location"`
	Picture struct {
		Large string `json:"large"`
	} `json:"picture"`
}

func fetchRandomUsers(gender string, count int) ([]randomUserResult, error) {
	url := fmt.Sprintf(
		"https://randomuser.me/api/?results=%d&gender=%s&nat=us,gb,fr,de,es,au&inc=name,dob,location,picture&noinfo",
		count, gender,
	)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("randomuser.me request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Results []randomUserResult `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("randomuser.me parse error: %w", err)
	}
	return payload.Results, nil
}

func (s *SeedService) EnsureMinimumUsers(ctx context.Context, minimum int) (int, int, error) {
	if minimum <= 0 {
		total, err := s.profileRepo.Count(ctx)
		return 0, total, err
	}

	totalProfiles, err := s.profileRepo.Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	if totalProfiles >= minimum {
		return 0, totalProfiles, nil
	}

	maleCount, femaleCount, err := s.profileRepo.CountByGender(ctx)
	if err != nil {
		return 0, totalProfiles, err
	}

	const targetPerGender = 250
	maleDeficit := maxInt(0, targetPerGender-maleCount)
	femaleDeficit := maxInt(0, targetPerGender-femaleCount)

	// also handle the case where minimum > 500
	extra := maxInt(0, minimum-totalProfiles-maleDeficit-femaleDeficit)
	maleDeficit += extra / 2
	femaleDeficit += extra - extra/2

	hash, err := bcrypt.GenerateFromPassword([]byte("SeedPassw0rd!"), bcrypt.DefaultCost)
	if err != nil {
		return 0, totalProfiles, err
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	created := 0

	if maleDeficit > 0 {
		males, err := fetchRandomUsers("male", maleDeficit)
		if err != nil {
			return created, totalProfiles + created, fmt.Errorf("fetch males: %w", err)
		}
		for _, u := range males {
			if err := s.createSeedUserFromAPI(ctx, hash, rng, u, "male"); err != nil {
				return created, totalProfiles + created, err
			}
			created++
		}
	}

	if femaleDeficit > 0 {
		females, err := fetchRandomUsers("female", femaleDeficit)
		if err != nil {
			return created, totalProfiles + created, fmt.Errorf("fetch females: %w", err)
		}
		for _, u := range females {
			if err := s.createSeedUserFromAPI(ctx, hash, rng, u, "female"); err != nil {
				return created, totalProfiles + created, err
			}
			created++
		}
	}

	return created, totalProfiles + created, nil
}

func (s *SeedService) createSeedUserFromAPI(ctx context.Context, hash []byte, rng *rand.Rand, ru randomUserResult, gender string) error {
	for {
		suffix := fmt.Sprintf("%06d", rng.Intn(1000000))
		username := "seed_" + strings.ToLower(ru.Name.First) + "_" + suffix
		email := username + "@matcha.local"

		userID := uuid.New()
		user := &repository.User{
			ID:           userID,
			Username:     username,
			Email:        email,
			PasswordHash: string(hash),
			FirstName:    ru.Name.First,
			LastName:     ru.Name.Last,
			EmailVerifiedAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				continue
			}
			return err
		}

		birthDate := time.Now().AddDate(-ru.Dob.Age, 0, 0)
		if ru.Dob.Date != "" {
			if t, err := time.Parse("2006-01-02T15:04:05.000Z", ru.Dob.Date); err == nil {
				birthDate = t
			}
		}

		city := ru.Location.City
		if city == "" {
			city, _, _ = randomLocation(rng)
		}
		lat, lon := parseCoord(ru.Location.Coordinates.Latitude), parseCoord(ru.Location.Coordinates.Longitude)
		// clamp to sane European-ish range if coords look wrong
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			_, lat, lon = randomLocation(rng)
		}

		preference := randomFrom(rng, []string{"male", "female", "both"})
		bio := fmt.Sprintf("Hi, I'm %s. I like meeting new people.", ru.Name.First)

		profile := &repository.Profile{
			UserID:           userID,
			Bio:              &bio,
			Gender:           &gender,
			SexualPreference: &preference,
			BirthDate:        &birthDate,
			City:             &city,
			Latitude:         &lat,
			Longitude:        &lon,
			FameRating:       0,
		}
		if err := s.profileRepo.Upsert(ctx, profile); err != nil {
			return err
		}

		tags := randomTags(rng)
		if err := s.profileRepo.SetTags(ctx, userID, tags); err != nil {
			return err
		}

		objectKey := fmt.Sprintf("seed/default/%s/01.jpg", userID.String())
		if _, err := s.photoRepo.Create(ctx, userID, objectKey, ru.Picture.Large, true); err != nil {
			return err
		}

		return nil
	}
}

func parseCoord(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func randomFrom(rng *rand.Rand, values []string) string {
	return values[rng.Intn(len(values))]
}

func randomLocation(rng *rand.Rand) (string, float64, float64) {
	type loc struct {
		city string
		lat  float64
		lon  float64
	}
	locations := []loc{
		{city: "Paris", lat: 48.8566, lon: 2.3522},
		{city: "Berlin", lat: 52.52, lon: 13.405},
		{city: "Madrid", lat: 40.4168, lon: -3.7038},
		{city: "Rome", lat: 41.9028, lon: 12.4964},
		{city: "Amsterdam", lat: 52.3676, lon: 4.9041},
	}
	l := locations[rng.Intn(len(locations))]
	return l.city, l.lat + (rng.Float64()-0.5)*0.1, l.lon + (rng.Float64()-0.5)*0.1
}

func randomTags(rng *rand.Rand) []string {
	pool := []string{"music", "travel", "books", "sports", "cinema", "art", "hiking", "food"}
	n := 2 + rng.Intn(3)
	picked := make([]string, 0, n)
	seen := map[string]struct{}{}
	for len(picked) < n {
		t := pool[rng.Intn(len(pool))]
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		picked = append(picked, t)
	}
	return picked
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
