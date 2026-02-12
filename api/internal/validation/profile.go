package validation

import (
	"fmt"
	"time"
)

const (
	MinAge = 18
	MaxBioLen = 500
)

var (
	GenderValues           = []string{"male", "female", "non-binary", "other"}
	SexualPreferenceValues = []string{"male", "female", "both", "other"}
)

func ValidateBirthDate(s string) error {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("birth_date: format YYYY-MM-DD (e.g. 1990-01-15)")
	}
	now := time.Now()
	if t.After(now) {
		return fmt.Errorf("birth_date: must be in the past, not future")
	}
	age := now.Year() - t.Year()
	if t.AddDate(age, 0, 0).After(now) {
		age--
	}
	if age < MinAge {
		return fmt.Errorf("birth_date: must be at least %d years old", MinAge)
	}
	return nil
}

func ValidateGender(s string) error {
	if s == "" {
		return nil
	}
	for _, v := range GenderValues {
		if s == v {
			return nil
		}
	}
	return fmt.Errorf("gender: must be one of %v", GenderValues)
}

func ValidateSexualPreference(s string) error {
	if s == "" {
		return nil
	}
	for _, v := range SexualPreferenceValues {
		if s == v {
			return nil
		}
	}
	return fmt.Errorf("sexual_preference: must be one of %v", SexualPreferenceValues)
}

func ValidateBio(s string) error {
	if len(s) > MaxBioLen {
		return fmt.Errorf("bio: max %d characters", MaxBioLen)
	}
	return nil
}

func ValidateLatitude(v float64) error {
	if v < -90 || v > 90 {
		return fmt.Errorf("latitude: must be between -90 and 90")
	}
	return nil
}

func ValidateLongitude(v float64) error {
	if v < -180 || v > 180 {
		return fmt.Errorf("longitude: must be between -180 and 180")
	}
	return nil
}
