//go:build integration

package redis_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	spicecache "github.com/StevenBuglione/spice/cache"
	redisstarter "github.com/StevenBuglione/spice/starter/redis"
)

type integrationValue struct {
	ID       string `json:"id"`
	Quantity int    `json:"quantity"`
}

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

	var observations []spicecache.Observation
	store, err := redisstarter.NewJSONStore[integrationValue](
		client,
		redisstarter.StoreOptions{
			Definition: spicecache.Definition{
				ID:     "integration.orders",
				Module: "example.com/integration/orders",
			},
			Prefix:        "spice-integration-orders",
			MaxValueBytes: 1024,
		},
		func(_ context.Context, observation spicecache.Observation) {
			observations = append(observations, observation)
		},
	)
	if err != nil {
		t.Fatalf("construct Redis cache: %v", err)
	}
	want := integrationValue{ID: "41", Quantity: 3}
	if err := store.Put(ctx, "41", want, 100*time.Millisecond); err != nil {
		t.Fatalf("put Redis cache entry: %v", err)
	}
	got, found, err := store.Get(ctx, "41")
	if err != nil {
		t.Fatalf("get Redis cache entry: %v", err)
	}
	if !found || got != want {
		t.Fatalf("cache Get() = %#v, %t; want %#v, true", got, found, want)
	}
	expirationDeadline := time.Now().Add(5 * time.Second)
	for found && time.Now().Before(expirationDeadline) {
		time.Sleep(25 * time.Millisecond)
		_, found, err = store.Get(ctx, "41")
		if err != nil {
			t.Fatalf("get expiring Redis cache entry: %v", err)
		}
	}
	if found {
		t.Fatal("Redis cache entry did not expire")
	}
	if err := store.Put(ctx, "42", want, 0); err != nil {
		t.Fatalf("put persistent Redis cache entry: %v", err)
	}
	if err := store.Delete(ctx, "42"); err != nil {
		t.Fatalf("delete Redis cache entry: %v", err)
	}
	if snapshot := store.Snapshot(); snapshot.Hits == 0 ||
		snapshot.Misses == 0 ||
		snapshot.Puts != 2 ||
		snapshot.Deletes != 1 {
		t.Fatalf("Redis cache snapshot = %#v", snapshot)
	}
	if len(observations) < 5 ||
		observations[0].Definition.Module !=
			"example.com/integration/orders" {
		t.Fatalf("Redis cache observations = %#v", observations)
	}

	canceled, cancelCanceled := context.WithCancel(context.Background())
	cancelCanceled()
	if _, _, err := store.Get(canceled, "41"); err == nil {
		t.Fatal("canceled Redis cache Get() unexpectedly succeeded")
	}
}
