package config

import "os"

func DatabaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://matcha:matcha_secret@localhost:5432/matcha?sslmode=disable"
}

func JWTSecret() string {
	if v := os.Getenv("JWT_SECRET"); v != "" {
		return v
	}
	return "dev_secret"
}

func CORSOrigin() string {
	if v := os.Getenv("CORS_ORIGIN"); v != "" {
		return v
	}
	return "http://localhost:3000"
}
