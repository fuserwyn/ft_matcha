package config

import "os"

func DatabaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://matcha:matcha_secret@localhost:5432/matcha?sslmode=disable"
}
