package validation

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	MinUsernameLen = 3
	MaxUsernameLen = 50
	MinPasswordLen = 8
	MaxPasswordLen = 72 // bcrypt limit
	MaxEmailLen    = 255
	MinNameLen     = 1
	MaxNameLen     = 100
)

// username: буквы, цифры, underscore
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func ValidateUsername(s string) error {
	s = strings.TrimSpace(s)
	if len(s) < MinUsernameLen {
		return fmt.Errorf("username: min %d characters", MinUsernameLen)
	}
	if len(s) > MaxUsernameLen {
		return fmt.Errorf("username: max %d characters", MaxUsernameLen)
	}
	if !usernameRegex.MatchString(s) {
		return fmt.Errorf("username: only letters, digits, underscore")
	}
	return nil
}

func ValidateEmail(s string) error {
	s = strings.TrimSpace(s)
	if len(s) > MaxEmailLen {
		return fmt.Errorf("email: max %d characters", MaxEmailLen)
	}
	return nil
}

func ValidatePassword(s string) error {
	if len(s) < MinPasswordLen {
		return fmt.Errorf("password: min %d characters", MinPasswordLen)
	}
	if len(s) > MaxPasswordLen {
		return fmt.Errorf("password: max %d characters", MaxPasswordLen)
	}
	return nil
}

func ValidateName(s string, field string) error {
	s = strings.TrimSpace(s)
	if len(s) < MinNameLen {
		return fmt.Errorf("%s: required", field)
	}
	if len(s) > MaxNameLen {
		return fmt.Errorf("%s: max %d characters", field, MaxNameLen)
	}
	return nil
}
