package handlers

import (
	"testing"
	"time"

	"matcha/api/internal/repository"
)

func TestIsDiscoveryProfileReady(t *testing.T) {
	now := time.Now().UTC()
	gender := "male"
	city := "Paris"

	ready := &repository.Profile{
		Gender:           &gender,
		SexualPreference: []string{"female"},
		BirthDate:        &now,
		City:             &city,
	}
	if !isDiscoveryProfileReady(ready) {
		t.Fatalf("expected ready profile to pass")
	}

	cases := []struct {
		name string
		p    *repository.Profile
	}{
		{"nil profile", nil},
		{"missing gender", &repository.Profile{SexualPreference: []string{"female"}, BirthDate: &now, City: &city}},
		{"blank gender", &repository.Profile{Gender: strPtr(" "), SexualPreference: []string{"female"}, BirthDate: &now, City: &city}},
		{"missing sexual preference", &repository.Profile{Gender: &gender, BirthDate: &now, City: &city}},
		{"missing birth date", &repository.Profile{Gender: &gender, SexualPreference: []string{"female"}, City: &city}},
		{"missing city", &repository.Profile{Gender: &gender, SexualPreference: []string{"female"}, BirthDate: &now}},
		{"blank city", &repository.Profile{Gender: &gender, SexualPreference: []string{"female"}, BirthDate: &now, City: strPtr(" ")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if isDiscoveryProfileReady(tc.p) {
				t.Fatalf("expected profile to be incomplete")
			}
		})
	}
}

func TestApplyDefaultDiscoveryGenders(t *testing.T) {
	f := &repository.DiscoveryFilters{}
	applyDefaultDiscoveryGenders(f)
	if len(f.Genders) != 2 || f.Genders[0] != "female" || f.Genders[1] != "male" {
		t.Fatalf("expected bisexual default [female male], got %#v", f.Genders)
	}

	f2 := &repository.DiscoveryFilters{Genders: []string{"non-binary"}}
	applyDefaultDiscoveryGenders(f2)
	if len(f2.Genders) != 1 || f2.Genders[0] != "non-binary" {
		t.Fatalf("expected explicit genders preserved, got %#v", f2.Genders)
	}
}

func strPtr(s string) *string { return &s }
