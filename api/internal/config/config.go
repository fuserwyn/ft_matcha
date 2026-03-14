package config

import (
	"os"
	"strconv"
	"strings"
)

func mustEnv(name string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		panic("missing required env: " + name)
	}
	return v
}

func DatabaseURL() string {
	return mustEnv("DATABASE_URL")
}

func RedisURL() string {
	return mustEnv("REDIS_URL")
}

func JWTSecret() string {
	return mustEnv("JWT_SECRET")
}

func CORSOrigin() string {
	if v := os.Getenv("CORS_ORIGIN"); v != "" {
		return v
	}
	return "http://localhost:3000"
}

func ElasticsearchURL() string {
	return mustEnv("ELASTICSEARCH_URL")
}

func MinIOEndpoint() string {
	return mustEnv("MINIO_ENDPOINT")
}

func MinIOAccessKey() string {
	return mustEnv("MINIO_ACCESS_KEY")
}

func MinIOSecretKey() string {
	return mustEnv("MINIO_SECRET_KEY")
}

func MinIOBucket() string {
	return mustEnv("MINIO_BUCKET")
}

func MinIOPublicBaseURL() string {
	if v := os.Getenv("MINIO_PUBLIC_BASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("MINIO_PUBLIC_URL"); v != "" {
		return v
	}
	return "http://localhost:9000"
}

func SMTPHost() string {
	return mustEnv("SMTP_HOST")
}

func SMTPPort() string {
	return mustEnv("SMTP_PORT")
}

func SMTPFrom() string {
	return mustEnv("SMTP_FROM")
}

func SMTPCooldownSeconds() int {
	if v := os.Getenv("SMTP_COOLDOWN_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 30
}

func PublicAPIBaseURL() string {
	if v := os.Getenv("PUBLIC_API_BASE_URL"); v != "" {
		return strings.TrimRight(strings.TrimSpace(v), "/")
	}
	return "http://localhost:8080"
}

func FrontendBaseURL() string {
	if v := os.Getenv("FRONTEND_BASE_URL"); v != "" {
		return strings.TrimRight(strings.TrimSpace(v), "/")
	}
	return "http://localhost:3000"
}

func SeedUsersEnabled() bool {
	if v := os.Getenv("SEED_USERS_ENABLED"); v != "" {
		return v == "1" || v == "true" || v == "TRUE"
	}
	return true
}

func MinUsersCount() int {
	if v := os.Getenv("MIN_USERS_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 500
}

func E2ESkipEmailVerification() bool {
	return os.Getenv("RUN_E2E") == "1"
}
