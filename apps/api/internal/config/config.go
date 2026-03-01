package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL               string
	RedisURL                  string
	Port                      string
	WorkerCount               int
	RecoveryIntervalSecs      int
	RecoveryIdleThresholdSecs int
	JWTSecret                 string
	AccessTokenTTL            time.Duration
	RefreshTokenTTL           time.Duration
}

func Load() Config {
	return Config{
		DatabaseURL:               getEnv("DATABASE_URL", "postgres://flowforge:flowforge@localhost:5432/flowforge?sslmode=disable"),
		RedisURL:                  getEnv("REDIS_URL", "redis://localhost:6379"),
		Port:                      getEnv("PORT", "8080"),
		WorkerCount:               getEnvInt("WORKER_COUNT", 5),
		RecoveryIntervalSecs:      getEnvInt("RECOVERY_INTERVAL_SECS", 30),
		RecoveryIdleThresholdSecs: getEnvInt("RECOVERY_IDLE_THRESHOLD_SECS", 60),
		JWTSecret:                 getEnv("JWT_SECRET", "dev-secret-change-me-in-prod"),
		AccessTokenTTL:            getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:           getEnvDuration("REFRESH_TOKEN_TTL", 720*time.Hour),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
