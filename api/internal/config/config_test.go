package config

import (
	"os"
	"strings"
	"testing"
)

func expectPanicContains(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, got nil", want)
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic type = %T, want string", r)
		}
		if !strings.Contains(msg, want) {
			t.Fatalf("panic = %q, want contains %q", msg, want)
		}
	}()
	fn()
}

func TestDatabaseURL(t *testing.T) {
	orig := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", orig)

	os.Setenv("DATABASE_URL", "postgres://custom:5432/db")
	if got := DatabaseURL(); got != "postgres://custom:5432/db" {
		t.Errorf("DatabaseURL() = %q, want postgres://custom:5432/db", got)
	}

	os.Unsetenv("DATABASE_URL")
	expectPanicContains(t, "missing required env: DATABASE_URL", func() { _ = DatabaseURL() })
}

func TestSMTPHost(t *testing.T) {
	orig := os.Getenv("SMTP_HOST")
	defer os.Setenv("SMTP_HOST", orig)

	os.Setenv("SMTP_HOST", "smtp.gmail.com")
	if got := SMTPHost(); got != "smtp.gmail.com" {
		t.Errorf("SMTPHost() = %q, want smtp.gmail.com", got)
	}

	os.Unsetenv("SMTP_HOST")
	expectPanicContains(t, "missing required env: SMTP_HOST", func() { _ = SMTPHost() })
}

func TestSMTPPort(t *testing.T) {
	orig := os.Getenv("SMTP_PORT")
	defer os.Setenv("SMTP_PORT", orig)

	os.Setenv("SMTP_PORT", "587")
	if got := SMTPPort(); got != "587" {
		t.Errorf("SMTPPort() = %q, want 587", got)
	}

	os.Unsetenv("SMTP_PORT")
	expectPanicContains(t, "missing required env: SMTP_PORT", func() { _ = SMTPPort() })
}

func TestSMTPFrom(t *testing.T) {
	orig := os.Getenv("SMTP_FROM")
	defer os.Setenv("SMTP_FROM", orig)

	os.Setenv("SMTP_FROM", "noreply@mydomain.com")
	if got := SMTPFrom(); got != "noreply@mydomain.com" {
		t.Errorf("SMTPFrom() = %q, want noreply@mydomain.com", got)
	}
}

func TestSMTPCooldownSeconds(t *testing.T) {
	orig := os.Getenv("SMTP_COOLDOWN_SECONDS")
	defer os.Setenv("SMTP_COOLDOWN_SECONDS", orig)

	os.Setenv("SMTP_COOLDOWN_SECONDS", "60")
	if got := SMTPCooldownSeconds(); got != 60 {
		t.Errorf("SMTPCooldownSeconds() = %d, want 60", got)
	}

	os.Setenv("SMTP_COOLDOWN_SECONDS", "invalid")
	if got := SMTPCooldownSeconds(); got != 30 {
		t.Errorf("SMTPCooldownSeconds() invalid = %d, want default 30", got)
	}
}
