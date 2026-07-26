package redis

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	spicecache "github.com/StevenBuglione/spice/cache"
)

type cachedOrder struct {
	ID       string `json:"id"`
	Quantity int    `json:"quantity"`
}

func TestJSONStoreImplementsTypedCacheWithKeyFreeObservations(t *testing.T) {
	t.Parallel()

	commands := newFakeCacheCommands()
	var observations []spicecache.Observation
	store, err := newJSONStore[cachedOrder](
		commands,
		StoreOptions{
			Definition: spicecache.Definition{
				ID:     "orders.by-id",
				Module: "example.com/shop/orders",
			},
			Prefix:        "orders-by-id",
			MaxValueBytes: 256,
		},
		func(_ context.Context, observation spicecache.Observation) {
			observations = append(observations, observation)
		},
	)
	if err != nil {
		t.Fatalf("newJSONStore() error = %v", err)
	}

	ctx := context.Background()
	want := cachedOrder{ID: "41", Quantity: 3}
	if putErr := store.Put(ctx, "41", want, time.Minute); putErr != nil {
		t.Fatalf("Put() error = %v", putErr)
	}
	if got := commands.value("orders-by-id:41"); string(got) !=
		`{"id":"41","quantity":3}` {
		t.Fatalf("encoded value = %q", got)
	}
	got, found, err := store.Get(ctx, "41")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found || got != want {
		t.Fatalf("Get() = %#v, %t; want %#v, true", got, found, want)
	}
	if err := store.Delete(ctx, "41"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, found, err := store.Get(ctx, "41"); err != nil || found {
		t.Fatalf("Get(deleted) found=%t error=%v", found, err)
	}

	if snapshot := store.Snapshot(); snapshot != (spicecache.Snapshot{
		Hits:    1,
		Misses:  1,
		Puts:    1,
		Deletes: 1,
	}) {
		t.Fatalf("Snapshot() = %#v", snapshot)
	}
	wantOperations := []spicecache.Operation{
		spicecache.OperationPut,
		spicecache.OperationGet,
		spicecache.OperationDelete,
		spicecache.OperationGet,
	}
	var operations []spicecache.Operation
	for _, observation := range observations {
		operations = append(operations, observation.Operation)
		if observation.Definition != (spicecache.Definition{
			ID:     "orders.by-id",
			Module: "example.com/shop/orders",
		}) || observation.Size != 0 {
			t.Fatalf("observation = %#v", observation)
		}
	}
	if len(observations) != len(wantOperations) {
		t.Fatalf("observations = %#v", observations)
	}
	if !slices.Equal(operations, wantOperations) ||
		!observations[1].Hit ||
		observations[2].Removed != 1 ||
		observations[3].Hit {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestNewJSONStoreValidatesConfiguration(t *testing.T) {
	t.Parallel()

	valid := StoreOptions{
		Definition: spicecache.Definition{
			ID:     "orders.by-id",
			Module: "example.com/shop/orders",
		},
		Prefix: "orders-by-id",
	}
	tests := []struct {
		name   string
		mutate func(*StoreOptions)
	}{
		{name: "cache ID", mutate: func(options *StoreOptions) {
			options.Definition.ID = ""
		}},
		{name: "module", mutate: func(options *StoreOptions) {
			options.Definition.Module = ""
		}},
		{name: "prefix empty", mutate: func(options *StoreOptions) {
			options.Prefix = ""
		}},
		{name: "prefix character", mutate: func(options *StoreOptions) {
			options.Prefix = "orders:"
		}},
		{name: "prefix long", mutate: func(options *StoreOptions) {
			options.Prefix = strings.Repeat("x", maxPrefixBytes+1)
		}},
		{name: "value limit negative", mutate: func(options *StoreOptions) {
			options.MaxValueBytes = -1
		}},
		{name: "value limit excess", mutate: func(options *StoreOptions) {
			options.MaxValueBytes = maxMaxValueBytes + 1
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			options := valid
			test.mutate(&options)
			if _, err := newJSONStore[any](
				newFakeCacheCommands(),
				options,
			); err == nil {
				t.Fatal("newJSONStore() unexpectedly succeeded")
			}
		})
	}

	store, err := newJSONStore[any](
		newFakeCacheCommands(),
		valid,
	)
	if err != nil {
		t.Fatalf("newJSONStore(default limit) error = %v", err)
	}
	if store.maxValueBytes != defaultMaxValueBytes {
		t.Fatalf(
			"maximum value bytes = %d, want %d",
			store.maxValueBytes,
			defaultMaxValueBytes,
		)
	}
	if _, err := newJSONStore[any](nil, valid); err == nil {
		t.Fatal("nil command client unexpectedly succeeded")
	}
	if _, err := newJSONStore[any](
		newFakeCacheCommands(),
		valid,
		nil,
	); err == nil {
		t.Fatal("nil observer unexpectedly succeeded")
	}
	if _, err := NewJSONStore[any](nil, valid); err == nil {
		t.Fatal("nil client unexpectedly succeeded")
	}
}

func TestJSONStoreValidatesOperationsBeforeRedis(t *testing.T) {
	t.Parallel()

	commands := newFakeCacheCommands()
	store := mustJSONStore[any](t, commands, 64)
	var nilStore *JSONStore[any]
	if _, _, err := nilStore.Get(context.Background(), "key"); err == nil {
		t.Fatal("nil Get() unexpectedly succeeded")
	}
	if err := nilStore.Put(context.Background(), "key", "value", 0); err == nil {
		t.Fatal("nil Put() unexpectedly succeeded")
	}
	if err := nilStore.Delete(context.Background(), "key"); err == nil {
		t.Fatal("nil Delete() unexpectedly succeeded")
	}
	if snapshot := nilStore.Snapshot(); snapshot != (spicecache.Snapshot{}) {
		t.Fatalf("nil Snapshot() = %#v", snapshot)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := store.Get(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(canceled) error = %v", err)
	}
	if err := store.Put(ctx, "key", "value", 0); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("Put(canceled) error = %v", err)
	}
	if err := store.Delete(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete(canceled) error = %v", err)
	}
	if _, _, err := store.Get(nilTestContext(), "key"); err == nil {
		t.Fatal("Get(nil context) unexpectedly succeeded")
	}
	if err := store.Put(nilTestContext(), "key", "value", 0); err == nil {
		t.Fatal("Put(nil context) unexpectedly succeeded")
	}
	if err := store.Delete(nilTestContext(), "key"); err == nil {
		t.Fatal("Delete(nil context) unexpectedly succeeded")
	}
	if err := store.Put(context.Background(), "key", "value", -time.Second); err == nil {
		t.Fatal("Put(negative TTL) unexpectedly succeeded")
	}

	invalidKeys := []string{
		"",
		"space key",
		"control\nkey",
		string([]byte{0xff}),
		strings.Repeat("x", maxKeyBytes+1),
	}
	for _, key := range invalidKeys {
		if _, _, err := store.Get(context.Background(), key); err == nil {
			t.Fatalf("Get(%q) unexpectedly succeeded", key)
		}
	}
	if commands.callCount() != 0 {
		t.Fatalf("invalid operations reached Redis %d time(s)", commands.callCount())
	}
}

func TestJSONStoreBoundsAndSafelyReportsEncodingFailures(t *testing.T) {
	t.Parallel()

	commands := newFakeCacheCommands()
	store := mustJSONStore[cachedOrder](t, commands, 16)
	commands.setValue("cache:oversized", []byte(strings.Repeat("x", 17)))
	if _, _, err := store.Get(
		context.Background(),
		"oversized",
	); err == nil || strings.Contains(err.Error(), strings.Repeat("x", 17)) {
		t.Fatalf("Get(oversized) error = %v", err)
	}
	commands.setValue("cache:invalid", []byte("{secret"))
	if _, _, err := store.Get(
		context.Background(),
		"invalid",
	); err == nil || strings.Contains(err.Error(), "{secret") {
		t.Fatalf("Get(invalid) error = %v", err)
	}
	if err := store.Put(
		context.Background(),
		"large",
		cachedOrder{ID: strings.Repeat("s", 32)},
		0,
	); err == nil || strings.Contains(err.Error(), strings.Repeat("s", 32)) {
		t.Fatalf("Put(large) error = %v", err)
	}

	functionStore := mustJSONStore[func()](t, commands, 64)
	if err := functionStore.Put(
		context.Background(),
		"function",
		func() {},
		0,
	); err == nil {
		t.Fatal("Put(function) unexpectedly succeeded")
	}
}

func TestJSONStorePreservesCommandFailuresWithoutKeys(t *testing.T) {
	t.Parallel()

	failure := errors.New("redis unavailable")
	tests := []struct {
		name string
		run  func(*JSONStore[string]) error
		set  func(*fakeCacheCommands)
	}{
		{
			name: "get",
			set:  func(commands *fakeCacheCommands) { commands.getErr = failure },
			run: func(store *JSONStore[string]) error {
				_, _, err := store.Get(context.Background(), "secret-key")
				return err
			},
		},
		{
			name: "put",
			set:  func(commands *fakeCacheCommands) { commands.setErr = failure },
			run: func(store *JSONStore[string]) error {
				return store.Put(
					context.Background(),
					"secret-key",
					"secret-value",
					0,
				)
			},
		},
		{
			name: "delete",
			set: func(commands *fakeCacheCommands) {
				commands.deleteErr = failure
			},
			run: func(store *JSONStore[string]) error {
				return store.Delete(context.Background(), "secret-key")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			commands := newFakeCacheCommands()
			test.set(commands)
			store := mustJSONStore[string](t, commands, 64)
			err := test.run(store)
			if !errors.Is(err, failure) {
				t.Fatalf("operation error = %v, want command failure", err)
			}
			if strings.Contains(err.Error(), "secret-key") ||
				strings.Contains(err.Error(), "secret-value") {
				t.Fatalf("operation error exposed cache data: %v", err)
			}
			if snapshot := store.Snapshot(); snapshot != (spicecache.Snapshot{}) {
				t.Fatalf("failed operation Snapshot() = %#v", snapshot)
			}
		})
	}
}

func TestJSONStoreCountsConcurrentOperations(t *testing.T) {
	t.Parallel()

	commands := newFakeCacheCommands()
	store := mustJSONStore[int](t, commands, 64)
	var wait sync.WaitGroup
	for index := range 32 {
		wait.Go(func() {
			key := strings.Repeat("x", index%8+1)
			if err := store.Put(
				context.Background(),
				key,
				index,
				0,
			); err != nil {
				t.Errorf("Put() error = %v", err)
			}
		})
	}
	wait.Wait()
	if snapshot := store.Snapshot(); snapshot.Puts != 32 {
		t.Fatalf("Snapshot().Puts = %d, want 32", snapshot.Puts)
	}
}

func mustJSONStore[V any](
	t *testing.T,
	commands cacheCommands,
	maximum int,
) *JSONStore[V] {
	t.Helper()
	store, err := newJSONStore[V](commands, StoreOptions{
		Definition: spicecache.Definition{
			ID:     "cache",
			Module: "example.com/module",
		},
		Prefix:        "cache",
		MaxValueBytes: maximum,
	})
	if err != nil {
		t.Fatalf("newJSONStore() error = %v", err)
	}
	return store
}

type fakeCacheCommands struct {
	mu        sync.Mutex
	entries   map[string][]byte
	getErr    error
	setErr    error
	deleteErr error
	calls     int
}

func newFakeCacheCommands() *fakeCacheCommands {
	return &fakeCacheCommands{entries: make(map[string][]byte)}
}

func (commands *fakeCacheCommands) getRange(
	_ context.Context,
	key string,
	maximum int64,
) ([]byte, error) {
	commands.mu.Lock()
	defer commands.mu.Unlock()
	commands.calls++
	if commands.getErr != nil {
		return nil, commands.getErr
	}
	value := commands.entries[key]
	if len(value) == 0 {
		return nil, nil
	}
	limit := min(len(value), int(maximum)+1)
	return append([]byte(nil), value[:limit]...), nil
}

func (commands *fakeCacheCommands) set(
	_ context.Context,
	key string,
	value []byte,
	_ time.Duration,
) error {
	commands.mu.Lock()
	defer commands.mu.Unlock()
	commands.calls++
	if commands.setErr != nil {
		return commands.setErr
	}
	commands.entries[key] = append([]byte(nil), value...)
	return nil
}

func (commands *fakeCacheCommands) delete(
	_ context.Context,
	key string,
) (bool, error) {
	commands.mu.Lock()
	defer commands.mu.Unlock()
	commands.calls++
	if commands.deleteErr != nil {
		return false, commands.deleteErr
	}
	_, found := commands.entries[key]
	delete(commands.entries, key)
	return found, nil
}

func (commands *fakeCacheCommands) value(key string) []byte {
	commands.mu.Lock()
	defer commands.mu.Unlock()
	return append([]byte(nil), commands.entries[key]...)
}

func (commands *fakeCacheCommands) setValue(key string, value []byte) {
	commands.mu.Lock()
	defer commands.mu.Unlock()
	commands.entries[key] = append([]byte(nil), value...)
}

func (commands *fakeCacheCommands) callCount() int {
	commands.mu.Lock()
	defer commands.mu.Unlock()
	return commands.calls
}
