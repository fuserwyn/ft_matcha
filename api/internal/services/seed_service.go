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
	return &SeedService{
		userRepo:    userRepo,
		profileRepo: profileRepo,
		photoRepo:   photoRepo,
	}
}

// ── Name pools ──────────────────────────────────────────────────────────────

var maleFirstNames = []string{
	"Liam", "Noah", "Oliver", "Elijah", "James", "William", "Benjamin", "Lucas",
	"Henry", "Alexander", "Mason", "Ethan", "Daniel", "Matthew", "Aiden", "Jackson",
	"Sebastian", "Jack", "Owen", "Samuel", "Ryan", "Nathan", "Leo", "Isaac",
	"Caleb", "Julian", "Ezra", "Adrian", "Nolan", "Thomas", "Finn", "Elliot",
	"Hugo", "Theo", "Oscar", "Felix", "Jasper", "Milo", "Archer", "Axel",
	"Rafael", "Mateo", "Santiago", "Andres", "Diego", "Luis", "Carlos", "Marco",
	"Antoine", "Pierre", "Jules", "Léo", "Baptiste", "Maxime", "Tristan", "Romain",
}

var femaleFirstNames = []string{
	"Olivia", "Emma", "Ava", "Charlotte", "Sophia", "Amelia", "Isabella", "Mia",
	"Evelyn", "Harper", "Luna", "Camila", "Gianna", "Elizabeth", "Eleanor", "Ella",
	"Abigail", "Sofia", "Avery", "Scarlett", "Emily", "Aria", "Penelope", "Chloe",
	"Layla", "Mila", "Nora", "Hazel", "Madison", "Ellie", "Lily", "Nova",
	"Isla", "Grace", "Violet", "Aurora", "Stella", "Zoey", "Natalie", "Hannah",
	"Sophie", "Alice", "Celine", "Léa", "Camille", "Lucie", "Manon", "Inès",
	"Valentina", "Elena", "Lucia", "Alba", "Martina", "Ana", "Clara", "Sara",
}

var nonBinaryFirstNames = []string{
	"Alex", "Jordan", "Morgan", "Taylor", "Riley", "Casey", "Jamie", "Quinn",
	"Avery", "Peyton", "Skyler", "Dakota", "Reese", "Finley", "Sage", "Blake",
	"River", "Phoenix", "Rowan", "Charlie", "Drew", "Emery", "Harper", "Hayden",
	"Jesse", "Kendall", "Lane", "Logan", "Marlowe", "Micah", "Parker", "Reagan",
	"Robin", "Sawyer", "Spencer", "Sterling", "Story", "Sydney", "Tatum", "Wynne",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Wilson", "Anderson", "Taylor", "Thomas", "Moore", "Jackson", "Martin", "Lee",
	"Thompson", "White", "Harris", "Clark", "Lewis", "Young", "Walker", "Hall",
	"Allen", "King", "Wright", "Scott", "Green", "Baker", "Adams", "Nelson",
	"Müller", "Schmidt", "Schneider", "Fischer", "Weber", "Meyer", "Wagner", "Becker",
	"Dupont", "Martin", "Bernard", "Moreau", "Simon", "Laurent", "Michel", "Garcia",
	"Rossi", "Ferrari", "Esposito", "Romano", "Colombo", "Ricci", "Marino", "Bruno",
}

// ── Location pool ────────────────────────────────────────────────────────────

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
	{"Stockholm", 59.3293, 18.0686},
	{"Copenhagen", 55.6761, 12.5683},
	{"Zurich", 47.3769, 8.5417},
	{"Prague", 50.0755, 14.4378},
	{"Milan", 45.4654, 9.1859},
}

// ── Bio templates ────────────────────────────────────────────────────────────

var maleBios = []string{
	"Outdoor enthusiast who loves hiking and photography. Looking for someone to share adventures with.",
	"Coffee addict and bookworm. I cook on weekends and pretend to be good at it.",
	"Musician by night, software engineer by day. Life is better with good music.",
	"Traveller at heart. Been to 30 countries and counting. Always planning the next trip.",
	"Gym regular who also loves lazy Sundays with a good movie and pizza.",
	"Passionate about sustainability and good food. Farmers market every Saturday.",
	"Dog dad, amateur chef, and weekend cyclist. Looking for genuine connection.",
	"Architecture lover, urban explorer. I find beauty in old buildings and quiet streets.",
	"Bookshelf full, heart open. Looking for someone to discuss big ideas with.",
	"Surfer when the waves allow, barista every morning. West coast soul.",
	"Aspiring novelist who reads too much and sleeps too little. Occasional runner.",
	"Ski instructor in winter, photographer in summer. Mountains are my home.",
	"Jazz fan and terrible dancer. Will always suggest going somewhere new.",
	"Introvert who loves deep conversations. Prefer small gatherings over big parties.",
	"Foodie with a soft spot for street food and hole-in-the-wall restaurants.",
}

var femaleBios = []string{
	"Yoga teacher and plant lover. My apartment is basically a jungle at this point.",
	"Art director who sketches everywhere. Always have a notebook, always observing.",
	"Marathon runner, terrible singer, excellent listener. Looking for real connection.",
	"Pastry chef who eats salads to compensate. Weekend hiker and cat mom.",
	"Literature PhD candidate. I quote books in everyday conversation — fair warning.",
	"Travel photographer based in Europe. Currently: everywhere. Always: curious.",
	"Dancer, dreamer, overthinker. Happiest near the ocean or in a good bookshop.",
	"Interior designer who loves vintage markets and mismatched furniture.",
	"Vegetarian cook, amateur sommelier. My dinner parties are legendary, apparently.",
	"Film buff and coffee snob. I can recommend a movie for any mood.",
	"Climber, reader, terrible at small talk but great at real talk.",
	"Freelance illustrator. I draw, travel, and spend too much in art supply stores.",
	"Nurse with a dark sense of humor. I need someone who can keep up.",
	"History nerd who visits every museum in every city I travel to.",
	"Minimalist trying to own fewer things, experience more. Work in progress.",
}

var nonBinaryBios = []string{
	"Queer artist navigating the world with curiosity and a bit of chaos.",
	"Non-binary, neurodivergent, and proud. I make music and bad puns.",
	"Writer and activist. Looking for depth, authenticity, and decent coffee.",
	"Gender-free zone. Into hiking, zines, community gardening, and good vibes.",
	"Photographer capturing queerness and joy. Life's too short for boring people.",
	"Librarian by day, poet by night. My pronouns are they/them.",
	"Tech worker who spends weekends at flea markets and art openings.",
	"Skateboarder, graphic novelist, aspiring chef. Loud opinions, open heart.",
	"Climate researcher with a passion for maps, bikes, and honest conversations.",
	"Musician exploring sound and identity. Always learning, always unlearning.",
	"Therapist-in-training who needs someone who does their own work too.",
	"Queer, vegan, chaotic good. I bake sourdough and read tarot for fun.",
	"Nonbinary traveller who collects languages and experiences, not things.",
	"Dancer and choreographer. Movement is my language, connection is my goal.",
	"Tattooed librarian with opinions about everything. Will talk philosophy at 2am.",
}

// ── Relationship goal pool ───────────────────────────────────────────────────

var relationshipGoals = []string{
	"long-term", "long-term-open", "short-term-open", "short-term", "friends", "not-sure",
}

// ── Tag pool ─────────────────────────────────────────────────────────────────

var tagPool = []string{
	"music", "travel", "books", "sports", "cinema", "art", "hiking", "food",
	"yoga", "gaming", "photography", "cooking", "dancing", "climbing", "cycling",
	"running", "coffee", "wine", "theatre", "concerts", "nature", "dogs", "cats",
	"surfing", "skiing", "reading", "writing", "tech", "design", "fashion",
}

// ── Sexual preference weights ────────────────────────────────────────────────

// Returns a realistic sexual preference array for the given gender.
func randomPreferences(rng *rand.Rand, gender string) []string {
	r := rng.Float64()
	switch gender {
	case "male":
		switch {
		case r < 0.55:
			return []string{"female"}
		case r < 0.70:
			return []string{"male", "female"}
		case r < 0.82:
			return []string{"male"}
		case r < 0.91:
			return []string{"female", "non-binary"}
		default:
			return []string{"male", "female", "non-binary", "other"}
		}
	case "female":
		switch {
		case r < 0.55:
			return []string{"male"}
		case r < 0.70:
			return []string{"male", "female"}
		case r < 0.82:
			return []string{"female"}
		case r < 0.91:
			return []string{"male", "non-binary"}
		default:
			return []string{"male", "female", "non-binary", "other"}
		}
	default: // non-binary
		switch {
		case r < 0.30:
			return []string{"male", "female", "non-binary"}
		case r < 0.55:
			return []string{"non-binary", "other"}
		case r < 0.70:
			return []string{"male", "female", "non-binary", "other"}
		case r < 0.83:
			return []string{"male"}
		default:
			return []string{"female"}
		}
	}
}

// ── Portrait photo pools (verified Unsplash CDN IDs, 400×600, crop=faces) ────

var femalePhotoIDs = []string{
	"1494790108377-be9c29b29330",
	"1534528741775-53994a69daeb",
	"1508214751196-bcfd4ca60f91",
	"1438761681033-6461ffad8d80",
	"1531746020798-e6953c6e8e04",
	"1554151228-14d9def656e4",
	"1517841905240-472988babdf9",
	"1488426862026-3ee34a7d66df",
	"1573496359142-b8d87734a5a2",
	"1540569014015-19a7be504e3a",
	"1494976388531-d1058494cdd8",
	"1490481651871-ab68de25d43d",
	"1607746882042-944635dfe10e",
	"1544005313-94ddf0286df2",
	"1520390138845-fd2d229dd553",
	"1589571894960-20bbe2828d0a",
	"1593104547489-5cfb3839a3b5",
	"1551836022-deb4988cc6c0",
	"1503252947848-7338d3f92f31",
	"1512316609839-ce289d3eba0a",
	"1524250502761-1ac6f2e30d43",
	"1529111290557-82f6d5c6cf85",
	"1556228578-8c89e6adf883",
	"1510227272981-87123e259b17",
	"1592621385612-4d7129426394",
}

var malePhotoIDs = []string{
	"1472099645785-5658abf4ff4e",
	"1500648767791-00dcc994a43e",
	"1507003211169-0a1dd7228f2d",
	"1506794778202-cad84cf45f1d",
	"1519085360753-af0119f7cbe7",
	"1492562080023-ab3db95bfbce",
	"1570295999919-56ceb5ecca61",
	"1568602471122-7832951cc4c5",
	"1534180477871-5d6cc81f3920",
	"1560250097-0b93528c311a",
	"1603415526960-f7e0328c63b1",
	"1559839914-17aae19cec71",
	"1622253692010-333f2da6031d",
	"1542178243-bc20204b769f",
	"1530268729831-4b0b9e170218",
	"1535713875002-d1d0cf377fde",
	"1581803118522-7b72a50f7e9f",
	"1569913486515-b74bf7751574",
	"1563178406-4cdc2923acbc",
	"1491555103944-7c647fd857e6",
	"1599566150163-29194dcaad36",
	"1560807707-8cc77767d783",
}

// ── Main entry point ─────────────────────────────────────────────────────────

const targetPerGender = 170

func (s *SeedService) EnsureMinimumUsers(ctx context.Context, minimum int) (int, int, error) {
	if minimum <= 0 {
		total, err := s.profileRepo.Count(ctx)
		return 0, total, err
	}

	maleCount, femaleCount, nonBinaryCount, err := s.profileRepo.CountByGender(ctx)
	if err != nil {
		return 0, 0, err
	}

	maleDeficit := maxInt(0, targetPerGender-maleCount)
	femaleDeficit := maxInt(0, targetPerGender-femaleCount)
	nonBinaryDeficit := maxInt(0, targetPerGender-nonBinaryCount)

	if maleDeficit == 0 && femaleDeficit == 0 && nonBinaryDeficit == 0 {
		total, err := s.profileRepo.Count(ctx)
		return 0, total, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("SeedPassw0rd!"), bcrypt.DefaultCost)
	if err != nil {
		return 0, 0, err
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	created := 0

	for i := 0; i < maleDeficit; i++ {
		if err := s.createSeedUser(ctx, hash, rng, "male", maleCount+i+1); err != nil {
			return created, 0, err
		}
		created++
	}
	for i := 0; i < femaleDeficit; i++ {
		if err := s.createSeedUser(ctx, hash, rng, "female", femaleCount+i+1); err != nil {
			return created, 0, err
		}
		created++
	}
	for i := 0; i < nonBinaryDeficit; i++ {
		if err := s.createSeedUser(ctx, hash, rng, "non-binary", nonBinaryCount+i+1); err != nil {
			return created, 0, err
		}
		created++
	}

	total, err := s.profileRepo.Count(ctx)
	return created, total, err
}

func (s *SeedService) createSeedUser(ctx context.Context, hash []byte, rng *rand.Rand, gender string, idx int) error {
	var firstName string
	switch gender {
	case "male":
		firstName = maleFirstNames[rng.Intn(len(maleFirstNames))]
	case "female":
		firstName = femaleFirstNames[rng.Intn(len(femaleFirstNames))]
	default:
		firstName = nonBinaryFirstNames[rng.Intn(len(nonBinaryFirstNames))]
	}
	lastName := lastNames[rng.Intn(len(lastNames))]

	genderSlug := strings.ReplaceAll(gender, "-", "")
	username := fmt.Sprintf("seed_%s_%s_%04d", genderSlug, strings.ToLower(firstName), idx)
	email := username + "@matcha.local"

	userID := uuid.New()
	user := &repository.User{
		ID:           userID,
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    firstName,
		LastName:     lastName,
		EmailVerifiedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("create user %s: %w", username, err)
	}

	// Age: 18–45
	ageYears := 18 + rng.Intn(28)
	birthDate := time.Now().AddDate(-ageYears, -rng.Intn(12), -rng.Intn(28))

	loc := europeanLocations[rng.Intn(len(europeanLocations))]
	city := loc.city
	lat := loc.lat + (rng.Float64()-0.5)*0.08
	lon := loc.lon + (rng.Float64()-0.5)*0.08

	preferences := randomPreferences(rng, gender)
	goal := relationshipGoals[rng.Intn(len(relationshipGoals))]

	var bio string
	switch gender {
	case "male":
		bio = maleBios[rng.Intn(len(maleBios))]
	case "female":
		bio = femaleBios[rng.Intn(len(femaleBios))]
	default:
		bio = nonBinaryBios[rng.Intn(len(nonBinaryBios))]
	}

	p := &repository.Profile{
		UserID:           userID,
		Bio:              &bio,
		Gender:           &gender,
		SexualPreference: preferences,
		RelationshipGoal: &goal,
		BirthDate:        &birthDate,
		City:             &city,
		Latitude:         &lat,
		Longitude:        &lon,
		FameRating:       0,
	}
	if err := s.profileRepo.Upsert(ctx, p); err != nil {
		return fmt.Errorf("upsert profile %s: %w", username, err)
	}

	tags := randomTags(rng)
	if err := s.profileRepo.SetTags(ctx, userID, tags); err != nil {
		return fmt.Errorf("set tags %s: %w", username, err)
	}

	// Portrait photo — verified Unsplash CDN (400×600, face-cropped)
	const unsplashBase = "https://images.unsplash.com/photo-%s?w=400&h=600&fit=crop&crop=faces&auto=format&q=80"
	var photoURL string
	switch gender {
	case "female":
		photoURL = fmt.Sprintf(unsplashBase, femalePhotoIDs[(idx-1)%len(femalePhotoIDs)])
	case "male":
		photoURL = fmt.Sprintf(unsplashBase, malePhotoIDs[(idx-1)%len(malePhotoIDs)])
	default:
		all := append(malePhotoIDs, femalePhotoIDs...)
		photoURL = fmt.Sprintf(unsplashBase, all[(idx-1)%len(all)])
	}
	objectKey := fmt.Sprintf("seed/%s/%s/01.jpg", gender, userID.String())
	if _, err := s.photoRepo.Create(ctx, userID, objectKey, photoURL, true); err != nil {
		return fmt.Errorf("create photo %s: %w", username, err)
	}

	return nil
}

func randomTags(rng *rand.Rand) []string {
	n := 2 + rng.Intn(4) // 2–5 tags
	picked := make([]string, 0, n)
	seen := map[string]struct{}{}
	for len(picked) < n {
		t := tagPool[rng.Intn(len(tagPool))]
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
