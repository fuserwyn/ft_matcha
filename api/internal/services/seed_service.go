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
	randomPhotoUsed map[string]struct{}
	malePhotoCount  int
	femalePhotoCount int
}

func NewSeedService(
	userRepo *repository.UserRepository,
	profileRepo *repository.ProfileRepository,
	photoRepo *repository.PhotoRepository,
) *SeedService {
	return &SeedService{
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		photoRepo:        photoRepo,
		randomPhotoUsed:  map[string]struct{}{},
		malePhotoCount:   0,
		femalePhotoCount: 0,
	}
}

type randomUserResult struct {
	Name struct {
		First string `json:"first"`
		Last  string `json:"last"`
	} `json:"name"`
	Dob struct {
		Date string `json:"date"`
		Age  int    `json:"age"`
	} `json:"dob"`
	Picture struct {
		Large string `json:"large"`
	} `json:"picture"`
}

const uniqueRandomUserPhotoPerGender = 89

func fetchRandomUsers(gender string, count int) ([]randomUserResult, error) {
	url := fmt.Sprintf(
		"https://randomuser.me/api/?results=%d&gender=%s&nat=us,gb,fr,de,es,au&inc=name,dob,picture&noinfo",
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

// europeanLocations are the cities used for all seed profiles so that
// distance/city filters work predictably during development.
var europeanLocations = []struct {
	city string
	lat  float64
	lon  float64
}{
	{"Paris", 48.8566, 2.3522},
	{"Berlin", 52.5200, 13.4050},
	{"Madrid", 40.4168, -3.7038},
	{"Rome", 41.9028, 12.4964},
	{"Amsterdam", 52.3676, 4.9041},
	{"Barcelona", 41.3851, 2.1734},
	{"Vienna", 48.2082, 16.3738},
	{"Munich", 48.1351, 11.5820},
	{"Brussels", 50.8503, 4.3517},
	{"Lisbon", 38.7169, -9.1395},
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

		// assign a random European city with slight coordinate jitter
		loc := europeanLocations[rng.Intn(len(europeanLocations))]
		city := loc.city
		lat := loc.lat + (rng.Float64()-0.5)*0.08
		lon := loc.lon + (rng.Float64()-0.5)*0.08

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
		url := s.pickSeedPhotoURL(userID, ru, gender)
		if _, err := s.photoRepo.Create(ctx, userID, objectKey, url, true); err != nil {
			return err
		}

		return nil
	}
}

func (s *SeedService) pickSeedPhotoURL(userID uuid.UUID, ru randomUserResult, gender string) string {
	portraitURL := strings.TrimSpace(ru.Picture.Large)
	normGender := strings.ToLower(strings.TrimSpace(gender))
	switch normGender {
	case "male":
		if s.malePhotoCount < uniqueRandomUserPhotoPerGender {
			if portraitURL != "" {
				if _, used := s.randomPhotoUsed[portraitURL]; !used {
					s.randomPhotoUsed[portraitURL] = struct{}{}
					s.malePhotoCount++
					return portraitURL
				}
			}
			if nextURL, ok := s.nextUnusedRandomUserPortraitURL("male"); ok {
				s.randomPhotoUsed[nextURL] = struct{}{}
				s.malePhotoCount++
				return nextURL
			}
		}
	case "female":
		if s.femalePhotoCount < uniqueRandomUserPhotoPerGender {
			if portraitURL != "" {
				if _, used := s.randomPhotoUsed[portraitURL]; !used {
					s.randomPhotoUsed[portraitURL] = struct{}{}
					s.femalePhotoCount++
					return portraitURL
				}
			}
			if nextURL, ok := s.nextUnusedRandomUserPortraitURL("female"); ok {
				s.randomPhotoUsed[nextURL] = struct{}{}
				s.femalePhotoCount++
				return nextURL
			}
		}
	}
	// Landscape fallback for all remaining seed users after 178 unique randomuser portraits.
	return fmt.Sprintf("https://picsum.photos/seed/landscape_%s/1200/800", userID.String())
}

func (s *SeedService) nextUnusedRandomUserPortraitURL(gender string) (string, bool) {
	path := "men"
	if gender == "female" {
		path = "women"
	}
	for i := 0; i < 100; i++ {
		candidate := fmt.Sprintf("https://randomuser.me/api/portraits/%s/%d.jpg", path, i)
		if _, used := s.randomPhotoUsed[candidate]; used {
			continue
		}
		return candidate, true
	}
	return "", false
}

func randomFrom(rng *rand.Rand, values []string) string {
	return values[rng.Intn(len(values))]
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
