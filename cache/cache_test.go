package cache

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryEvictsLeastRecentlyUsedEntry(t *testing.T) {
	t.Parallel()
	var observations []Observation
	memory := newTestMemory(t, 2, nil, func(_ context.Context, observation Observation) {
		observations = append(observations, observation)
	})
	ctx := context.Background()
	mustPut(t, memory, ctx, "a", "alpha", 0)
	mustPut(t, memory, ctx, "b", "bravo", 0)
	if value, found, err := memory.Get(ctx, "a"); err != nil || !found || value != "alpha" {
		t.Fatalf("Get(a) = %q, %v, %v", value, found, err)
	}
	mustPut(t, memory, ctx, "c", "charlie", 0)
	if _, found, err := memory.Get(ctx, "b"); err != nil || found {
		t.Fatalf("Get(b) found = %v, error = %v", found, err)
	}
	if value, found, err := memory.Get(ctx, "c"); err != nil || !found || value != "charlie" {
		t.Fatalf("Get(c) = %q, %v, %v", value, found, err)
	}
	snapshot := memory.Snapshot()
	if snapshot != (Snapshot{Size: 2, Hits: 2, Misses: 1, Puts: 3, Evictions: 1}) {
		t.Fatalf("Snapshot() = %#v", snapshot)
	}
	if len(observations) != 6 ||
		observations[3].Operation != OperationPut ||
		observations[3].Evicted != 1 ||
		observations[3].Definition != cacheDefinition() {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestMemoryExpiresAndPurgesWithCallerClock(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000, 0)
	var clockMu sync.Mutex
	clock := func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return now
	}
	setTime := func(next time.Time) {
		clockMu.Lock()
		now = next
		clockMu.Unlock()
	}
	memory := newTestMemory(t, 3, clock)
	ctx := context.Background()
	mustPut(t, memory, ctx, "short", "one", time.Second)
	mustPut(t, memory, ctx, "long", "two", time.Minute)
	mustPut(t, memory, ctx, "forever", "three", 0)

	setTime(now.Add(time.Second))
	if _, found, err := memory.Get(ctx, "short"); err != nil || found {
		t.Fatalf("Get(expired) found = %v, error = %v", found, err)
	}
	setTime(now.Add(time.Minute))
	removed, err := memory.PurgeExpired(ctx)
	if err != nil || removed != 1 {
		t.Fatalf("PurgeExpired() = %d, %v", removed, err)
	}
	if value, found, err := memory.Get(ctx, "forever"); err != nil || !found || value != "three" {
		t.Fatalf("Get(forever) = %q, %v, %v", value, found, err)
	}
	if snapshot := memory.Snapshot(); snapshot.Expired != 2 || snapshot.Size != 1 {
		t.Fatalf("Snapshot() = %#v", snapshot)
	}
}

func TestMemoryReplacesAndDeletesEntries(t *testing.T) {
	t.Parallel()
	memory := newTestMemory(t, 1, nil)
	ctx := context.Background()
	mustPut(t, memory, ctx, "key", "old", 0)
	mustPut(t, memory, ctx, "key", "new", 0)
	if snapshot := memory.Snapshot(); snapshot.Evictions != 0 || snapshot.Puts != 2 {
		t.Fatalf("Snapshot() = %#v", snapshot)
	}
	if err := memory.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := memory.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete(missing) error = %v", err)
	}
	if snapshot := memory.Snapshot(); snapshot.Deletes != 1 || snapshot.Size != 0 {
		t.Fatalf("Snapshot() = %#v", snapshot)
	}
}

func TestMemoryValidatesInputsAndCancellation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		definition Definition
		capacity   int
		observers  []Observer
	}{
		{"missing ID", Definition{Module: "example.com/shop"}, 1, nil},
		{"missing module", Definition{ID: "orders"}, 1, nil},
		{"invalid capacity", cacheDefinition(), 0, nil},
		{"nil observer", cacheDefinition(), 1, []Observer{nil}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewMemory[string, string](
				test.definition,
				test.capacity,
				nil,
				test.observers...,
			); err == nil {
				t.Fatal("NewMemory() error = nil")
			}
		})
	}

	memory := newTestMemory(t, 1, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := memory.Get(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(canceled) error = %v", err)
	}
	if err := memory.Put(ctx, "key", "value", 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("Put(canceled) error = %v", err)
	}
	if err := memory.Delete(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete(canceled) error = %v", err)
	}
	if _, err := memory.PurgeExpired(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("PurgeExpired(canceled) error = %v", err)
	}
	if err := memory.Put(context.Background(), "key", "value", -1); err == nil {
		t.Fatal("Put(negative TTL) error = nil")
	}
}

func TestNilMemoryAndContextAreSafe(t *testing.T) {
	t.Parallel()
	var memory *Memory[string, string]
	if snapshot := memory.Snapshot(); snapshot != (Snapshot{}) {
		t.Fatalf("nil Snapshot() = %#v", snapshot)
	}
	if _, _, err := memory.Get(context.Background(), "key"); err == nil {
		t.Fatal("nil Get() error = nil")
	}
	if err := memory.Put(context.Background(), "key", "value", 0); err == nil {
		t.Fatal("nil Put() error = nil")
	}
	if err := memory.Delete(context.Background(), "key"); err == nil {
		t.Fatal("nil Delete() error = nil")
	}
	if _, err := memory.PurgeExpired(context.Background()); err == nil {
		t.Fatal("nil PurgeExpired() error = nil")
	}
	initialized := newTestMemory(t, 1, nil)
	if _, _, err := initialized.Get(nilTestContext(), "key"); err == nil {
		t.Fatal("Get(nil context) error = nil")
	}
	if err := initialized.Put(nilTestContext(), "key", "value", 0); err == nil {
		t.Fatal("Put(nil context) error = nil")
	}
	if err := initialized.Delete(nilTestContext(), "key"); err == nil {
		t.Fatal("Delete(nil context) error = nil")
	}
	if _, err := initialized.PurgeExpired(nilTestContext()); err == nil {
		t.Fatal("PurgeExpired(nil context) error = nil")
	}
}

func TestMemorySupportsConcurrentAccess(t *testing.T) {
	t.Parallel()
	memory := newTestMemory(t, 64, nil)
	var failures atomic.Int64
	const workers = 64
	var wait sync.WaitGroup
	wait.Add(workers)
	for index := range workers {
		go func() {
			defer wait.Done()
			ctx := context.Background()
			key := string(rune('A' + index))
			if err := memory.Put(ctx, key, key, 0); err != nil {
				failures.Add(1)
				return
			}
			if value, found, err := memory.Get(ctx, key); err != nil || !found || value != key {
				failures.Add(1)
			}
		}()
	}
	wait.Wait()
	if got := failures.Load(); got != 0 {
		t.Fatalf("concurrent failures = %d", got)
	}
	if snapshot := memory.Snapshot(); snapshot.Size != workers || snapshot.Hits != workers {
		t.Fatalf("Snapshot() = %#v", snapshot)
	}
}

func newTestMemory(
	t *testing.T,
	capacity int,
	clock func() time.Time,
	observers ...Observer,
) *Memory[string, string] {
	t.Helper()
	memory, err := NewMemory[string, string](cacheDefinition(), capacity, clock, observers...)
	if err != nil {
		t.Fatalf("NewMemory() error = %v", err)
	}
	return memory
}

func mustPut(
	t *testing.T,
	memory *Memory[string, string],
	ctx context.Context,
	key string,
	value string,
	ttl time.Duration,
) {
	t.Helper()
	if err := memory.Put(ctx, key, value, ttl); err != nil {
		t.Fatalf("Put(%q) error = %v", key, err)
	}
}

func cacheDefinition() Definition {
	return Definition{ID: "orders.by-id", Module: "example.com/shop/orders"}
}

func nilTestContext() context.Context {
	return nil
}

func TestOperationsRemainStable(t *testing.T) {
	t.Parallel()
	if got := []Operation{OperationGet, OperationPut, OperationDelete, OperationPurge}; !slices.Equal(
		got,
		[]Operation{"get", "put", "delete", "purge"},
	) {
		t.Fatalf("operations = %v", got)
	}
}
