package validation

import (
	"strings"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"ok", "user123", false},
		{"ok underscore", "user_name", false},
		{"ok mixed case", "User_Name_1", false},
		{"min len", "abc", false},
		{"too short", "ab", true},
		{"empty", "", true},
		{"whitespace", "  user  ", false},
		{"invalid chars", "user@name", true},
		{"invalid hyphen", "user-name", true},
		{"max len", strings.Repeat("a", MaxUsernameLen), false},
		{"over max", strings.Repeat("a", MaxUsernameLen+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername(%q) err=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"ok", "SecurePass1", false},
		{"ok symbols", "MyP@ssw0rd!", false},
		{"too short", "short", true},
		{"common", "password", true},
		{"common 123", "12345678", true},
		{"common qwerty", "qwerty123", true},
		{"common admin", "admin", true},
		{"common welcome", "welcome", true},
		{"max len", strings.Repeat("a", MaxPasswordLen), false},
		{"over max", strings.Repeat("a", MaxPasswordLen+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) err=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"ok", "user@example.com", false},
		{"empty", "", false},
		{"over max", strings.Repeat("a", MaxEmailLen+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) err=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		field   string
		wantErr bool
	}{
		{"ok", "John", "first_name", false},
		{"empty", "", "first_name", true},
		{"over max", strings.Repeat("a", MaxNameLen+1), "first_name", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input, tt.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) err=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}
