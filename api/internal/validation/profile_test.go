package validation

import (
	"strings"
	"testing"
	"time"
)

func TestValidateBirthDate(t *testing.T) {
	now := time.Now()
	validPast := now.AddDate(-25, 0, 0).Format("2006-01-02")
	tooYoung := now.AddDate(-17, 0, 0).Format("2006-01-02")
	future := now.AddDate(1, 0, 0).Format("2006-01-02")

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"ok", validPast, false},
		{"empty", "", false},
		{"too young", tooYoung, true},
		{"future", future, true},
		{"bad format", "not-a-date", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBirthDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBirthDate(%q) err=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateGender(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"male", "male", false},
		{"female", "female", false},
		{"non-binary", "non-binary", false},
		{"other", "other", false},
		{"empty", "", false},
		{"invalid", "invalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGender(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGender(%q) err=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSexualPreference(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		wantErr bool
	}{
		{"single male", []string{"male"}, false},
		{"single female", []string{"female"}, false},
		{"single other", []string{"other"}, false},
		{"multiple", []string{"male", "female"}, false},
		{"empty", []string{}, false},
		{"invalid", []string{"invalid"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSexualPreference(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSexualPreference(%v) err=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLatitude(t *testing.T) {
	tests := []struct {
		name    string
		input   float64
		wantErr bool
	}{
		{"ok", 48.85, false},
		{"zero", 0, false},
		{"min", -90, false},
		{"max", 90, false},
		{"over max", 91, true},
		{"under min", -91, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLatitude(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLatitude(%v) err=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLongitude(t *testing.T) {
	tests := []struct {
		name    string
		input   float64
		wantErr bool
	}{
		{"ok", 2.35, false},
		{"min", -180, false},
		{"max", 180, false},
		{"over max", 181, true},
		{"under min", -181, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLongitude(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLongitude(%v) err=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBio(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"ok", "Hello world", false},
		{"empty", "", false},
		{"max len", strings.Repeat("a", MaxBioLen), false},
		{"over max", strings.Repeat("a", MaxBioLen+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBio(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBio err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCity(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"ok", "Paris", false},
		{"empty", "", false},
		{"max len", strings.Repeat("a", MaxCityLen), false},
		{"over max", strings.Repeat("a", MaxCityLen+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCity(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCity err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTags(t *testing.T) {
	overCount := make([]string, MaxTagsCount+1)
	for i := range overCount {
		overCount[i] = "tag"
	}
	tests := []struct {
		name    string
		input   []string
		wantErr bool
	}{
		{"ok", []string{"music", "travel"}, false},
		{"empty", []string{}, false},
		{"empty tag", []string{"music", ""}, true},
		{"over count", overCount, true},
		{"over len", []string{strings.Repeat("a", MaxTagLen+1)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTags(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTags err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
