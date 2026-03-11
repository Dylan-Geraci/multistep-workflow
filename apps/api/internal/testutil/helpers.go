package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/dylangeraci/flowforge/internal/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func SetupTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set, skipping integration test")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("Invalid TEST_REDIS_URL: %v", err)
	}
	rdb := redis.NewClient(opts)
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

func AuthContext(userID string) context.Context {
	return middleware.SetUserIDForTest(context.Background(), userID)
}

func CreateTestUser(t *testing.T, pool *pgxpool.Pool, userID, email string) {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password_hash, display_name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, now(), now()) ON CONFLICT (email) DO NOTHING`,
		userID, email, string(hash), fmt.Sprintf("Test User %s", userID),
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
}
