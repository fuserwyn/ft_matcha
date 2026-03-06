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
		if err := s.createDefaultPhotos(ctx, user.ID, gender, rng); err != nil {
			return err
		}
		return nil
	}
}

// High-quality portrait photo IDs from Unsplash CDN (600×800, face-cropped)
// All IDs verified HTTP 200 and confirmed correct gender
var unsplashMale = []string{
	"1507003211169-0a1dd7228f2d", "1472099645785-5658abf4ff4e", "1463453091185-61582044d556",
	"1504257432389-52343af06ae3", "1508214751196-bcfd4ca60f91", "1500048993953-d23a436266cf",
	"1539571696357-5a69c17a67c6", "1547425260-76bcadfb4f2c", "1560250097-0b93528c311a",
	"1568602471122-7832951cc4c5", "1552374196-1ab2a1c593e8", "1492562080023-ab3db95bfbce",
	"1580273916550-e323be2ae537", "1603415526960-f7e0328c63b1", "1554151228-14d9def656e4",
	"1564564321837-a57b7070ac4f", "1509909756405-be0199881695", "1499996860823-5214fcc65f8f",
	"1573496359142-b8d87734a5a2", "1599566150163-29194dcaad36",
}

var unsplashFemale = []string{
	"1438761681033-6461ffad8d80", "1487412720507-e7ab37603c6f", "1534528741775-53994a69daeb",
	"1517841905240-472988babdf9", "1520813792240-56fc4a3765a7", "1494790108377-be9c29b29330",
	"1502823403499-6ccfcf4fb453", "1540569014015-19a7be504e3a", "1519699047748-de8e457a634e",
	"1532074205216-d0e1f4b87368", "1594824476967-48c8b964273f", "1529626455594-4ff0802cfb7e",
	"1557555187-23d685287bc3", "1604004555489-723a93d6ce74", "1519345182560-3f2917c472ef",
	"1596451190630-186aff535bf2", "1610389051254-64849803c8fd", "1535713875002-d1d0cf377fde",
	"1541271696563-3be2f555fc4e", "1580489944761-15a19d654956", "1559598467-f8b76c8155d0",
	"1619895862022-09114b41f16f",
}

func unsplashPortraitURL(id string) string {
	return fmt.Sprintf("https://images.unsplash.com/photo-%s?w=600&h=800&fit=crop&crop=faces&auto=format&q=80", id)
}

func (s *SeedService) createDefaultPhotos(ctx context.Context, userID uuid.UUID, gender string, rng *rand.Rand) error {
	pool := unsplashMale
	if gender == "female" {
		pool = unsplashFemale
	}
	photoCount := 1 + rng.Intn(2)
	used := map[int]bool{}
	for i := 0; i < photoCount; i++ {
		n := rng.Intn(len(pool))
		for used[n] {
			n = rng.Intn(len(pool))
		}
		used[n] = true
		objectKey := fmt.Sprintf("seed/default/%s/%02d.jpg", userID.String(), i+1)
		if _, err := s.photoRepo.Create(ctx, userID, objectKey, unsplashPortraitURL(pool[n]), i == 0); err != nil {
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
