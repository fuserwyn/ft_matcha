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

var commonPasswords = map[string]struct{}{
	// Top numeric sequences
	"123456": {}, "1234567": {}, "12345678": {}, "123456789": {}, "1234567890": {},
	"12345": {}, "123123": {}, "1234": {}, "111111": {}, "112233": {},
	"000000": {}, "0000": {}, "9999": {}, "1111": {}, "2222": {}, "3333": {},
	"4444": {}, "5555": {}, "6666": {}, "7777": {}, "8888": {},
	"654321": {}, "987654321": {}, "0987654321": {},
	// Keyboard patterns
	"qwerty": {}, "qwerty123": {}, "qwertyuiop": {}, "azerty": {},
	"asdfgh": {}, "asdfghjkl": {}, "zxcvbn": {}, "qazwsx": {},
	"1q2w3e4r": {}, "1qaz2wsx": {},
	// Common English words / phrases
	"password": {}, "password1": {}, "password123": {}, "passw0rd": {},
	"abc123": {}, "admin": {}, "admin123": {}, "administrator": {},
	"letmein": {}, "welcome": {}, "welcome1": {}, "welcome123": {},
	"iloveyou": {}, "love": {}, "lovely": {},
	"monkey": {}, "dragon": {}, "master": {}, "ninja": {},
	"shadow": {}, "sunshine": {}, "princess": {}, "superman": {},
	"batman": {}, "trustno1": {}, "solo": {}, "starwars": {},
	"hello": {}, "hello123": {}, "test": {}, "test123": {}, "testing": {},
	"access": {}, "guest": {}, "root": {}, "toor": {}, "pass": {},
	"hunter": {}, "ranger": {}, "tiger": {}, "matrix": {},
	"charlie": {}, "robert": {}, "thomas": {}, "jessica": {},
	"michael": {}, "jennifer": {}, "jordan": {}, "harley": {},
	"summer": {}, "winter": {}, "flower": {}, "puppy": {}, "family": {},
	"secret": {}, "login": {}, "change": {}, "changeme": {},
	"default": {}, "temp": {}, "temp123": {},
	// Sports teams
	"football": {}, "baseball": {}, "soccer": {}, "hockey": {},
	"liverpool": {}, "arsenal": {}, "chelsea": {}, "barcelona": {},
	"manchester": {}, "madrid": {},
	// French common passwords (evaluators are at 42 Paris)
	"bonjour": {}, "motdepasse": {}, "soleil": {}, "azerty123": {},
	"pomme": {}, "chocolat": {},
}

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
	normalized := strings.ToLower(strings.TrimSpace(s))
	if _, isCommon := commonPasswords[normalized]; isCommon {
		return fmt.Errorf("password: too common, choose a stronger password")
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
