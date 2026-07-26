//go:build integration

package redis_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	redisstarter "github.com/StevenBuglione/spice/starter/redis"
)

func TestRedisClientIntegration(t *testing.T) {
	connectionURL := os.Getenv("SPICE_REDIS_TEST_URL")
	if connectionURL == "" {
		t.Fatal("SPICE_REDIS_TEST_URL is required for integration tests")
	}
	client, cleanup, err := redisstarter.Open(redisstarter.Options{
		URL:                  connectionURL,
		PoolSize:             4,
		AllowInsecure:        strings.HasPrefix(connectionURL, "redis://"),
		AllowUnauthenticated: true,
	})
	if err != nil {
		t.Fatalf("open Redis client: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()
		if cleanupErr := cleanup(cleanupContext); cleanupErr != nil {
			t.Errorf("cleanup Redis client: %v", cleanupErr)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("ping Redis client: %v", err)
	}
}
