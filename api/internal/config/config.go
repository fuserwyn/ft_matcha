package config

import (
	"os"
	"strconv"
)

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

func ElasticsearchURL() string {
	if v := os.Getenv("ELASTICSEARCH_URL"); v != "" {
		return v
	}
	return "http://localhost:9200"
}

func MinIOEndpoint() string {
	if v := os.Getenv("MINIO_ENDPOINT"); v != "" {
		return v
	}
	return "localhost:9000"
}

func MinIOAccessKey() string {
	if v := os.Getenv("MINIO_ACCESS_KEY"); v != "" {
		return v
	}
	return "minioadmin"
}

func MinIOSecretKey() string {
	if v := os.Getenv("MINIO_SECRET_KEY"); v != "" {
		return v
	}
	return "minioadmin"
}

func MinIOBucket() string {
	if v := os.Getenv("MINIO_BUCKET"); v != "" {
		return v
	}
	return "matcha-photos"
}

func SMTPHost() string {
	if v := os.Getenv("SMTP_HOST"); v != "" {
		return v
	}
	return "localhost"
}

func SMTPPort() string {
	if v := os.Getenv("SMTP_PORT"); v != "" {
		return v
	}
	return "1025"
}

func SMTPFrom() string {
	if v := os.Getenv("SMTP_FROM"); v != "" {
		return v
	}
	return "noreply@matcha.local"
}

func SMTPCooldownSeconds() int {
	if v := os.Getenv("SMTP_COOLDOWN_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 30
}
