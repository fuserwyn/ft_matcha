package services

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
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

	hash, err := bcrypt.GenerateFromPassword([]byte("SeedPassw0rd!"), bcrypt.DefaultCost)
	if err != nil {
		return 0, totalProfiles, err
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	created := 0
	totalUsers, err := s.userRepo.Count(ctx)
	if err != nil {
		return 0, totalProfiles, err
	}
	maleCount, femaleCount, err := s.profileRepo.CountByGender(ctx)
	if err != nil {
		return 0, totalProfiles, err
	}

	const targetPerGender = 250
	maleDeficit := maxInt(0, targetPerGender-maleCount)
	femaleDeficit := maxInt(0, targetPerGender-femaleCount)

	for i := 0; i < maleDeficit; i++ {
		if err := s.createSeedUser(ctx, hash, rng, totalUsers+created+1, "male"); err != nil {
			return created, totalProfiles + created, err
		}
		created++
		maleCount++
	}
	for i := 0; i < femaleDeficit; i++ {
		if err := s.createSeedUser(ctx, hash, rng, totalUsers+created+1, "female"); err != nil {
			return created, totalProfiles + created, err
		}
		created++
		femaleCount++
	}

	for totalProfiles+created < minimum {
		gender := "male"
		if femaleCount < maleCount {
			gender = "female"
		}
		if err := s.createSeedUser(ctx, hash, rng, totalUsers+created+1, gender); err != nil {
			return created, totalProfiles + created, err
		}
		created++
		if gender == "male" {
			maleCount++
		} else {
			femaleCount++
		}
	}

	return created, totalProfiles + created, nil
}

func (s *SeedService) createSeedUser(ctx context.Context, hash []byte, rng *rand.Rand, n int, gender string) error {
	for {
		suffix := fmt.Sprintf("%04d_%03d", n, rng.Intn(1000))
		username := "seed_user_" + suffix
		email := "seed_user_" + suffix + "@matcha.local"

		user := &repository.User{
			ID:           uuid.New(),
			Username:     username,
			Email:        email,
			PasswordHash: string(hash),
			FirstName:    randomFirstNameByGender(rng, gender),
			LastName:     randomLastName(rng),
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

		city, lat, lon := randomLocation(rng)
		preference := randomFrom(rng, []string{"male", "female", "both"})
		birthDate := time.Now().AddDate(-(18 + rng.Intn(28)), 0, -rng.Intn(365))
		bio := fmt.Sprintf("Hi, I'm %s. I like meeting new people.", user.FirstName)

		profile := &repository.Profile{
			UserID:           user.ID,
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
		if err := s.profileRepo.SetTags(ctx, user.ID, tags); err != nil {
			return err
		}
		if err := s.createDefaultPhotos(ctx, user.ID, user.Username, rng); err != nil {
			return err
		}
		return nil
	}
}

func (s *SeedService) createDefaultPhotos(ctx context.Context, userID uuid.UUID, username string, rng *rand.Rand) error {
	photoCount := 1 + rng.Intn(2)
	for i := 0; i < photoCount; i++ {
		objectKey := fmt.Sprintf("seed/default/%s/%02d.jpg", userID.String(), i+1)
		seed := fmt.Sprintf("%s_%02d", username, i+1)
		url := fmt.Sprintf("https://picsum.photos/seed/%s/600/800", seed)
		if _, err := s.photoRepo.Create(ctx, userID, objectKey, url, i == 0); err != nil {
			return err
		}
	}
	return nil
}

func randomFrom(rng *rand.Rand, values []string) string {
	return values[rng.Intn(len(values))]
}

func randomFirstNameByGender(rng *rand.Rand, gender string) string {
	switch gender {
	case "male":
		return randomFrom(rng, []string{"Alex", "Sam", "Chris", "Leo", "Noah", "Max", "Liam", "Ethan"})
	case "female":
		return randomFrom(rng, []string{"Nina", "Mia", "Eva", "Anna", "Olivia", "Emma", "Luna", "Sofia"})
	default:
		return randomFrom(rng, []string{"Alex", "Sam", "Chris", "Nina", "Mia", "Leo", "Eva", "Noah"})
	}
}

func randomLastName(rng *rand.Rand) string {
	return randomFrom(rng, []string{"Smith", "Taylor", "Brown", "Miller", "Wilson", "Moore", "Davis"})
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
