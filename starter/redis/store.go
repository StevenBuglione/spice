package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	redisclient "github.com/redis/go-redis/v9"
	spicecache "github.com/spice-framework/spice/cache"
)

const (
	defaultMaxValueBytes = 1 << 20
	maxMaxValueBytes     = 64 << 20
	maxPrefixBytes       = 128
	maxKeyBytes          = 512
	maxRedisKeyBytes     = 1024
)

// StoreOptions defines one namespaced typed Redis cache.
type StoreOptions struct {
	Definition    spicecache.Definition
	Prefix        string
	MaxValueBytes int
}

// JSONStore is a typed Redis cache that encodes values as JSON. It never
// exposes keys or values through observations or errors.
type JSONStore[V any] struct {
	commands      cacheCommands
	definition    spicecache.Definition
	prefix        string
	maxValueBytes int
	observers     []spicecache.Observer

	mu    sync.Mutex
	stats spicecache.Snapshot
}

type cacheCommands interface {
	getRange(context.Context, string, int64) ([]byte, error)
	set(context.Context, string, []byte, time.Duration) error
	delete(context.Context, string) (bool, error)
}

type nativeCacheCommands struct {
	client *redisclient.Client
}

// NewJSONStore constructs a typed, namespaced Redis cache. The client retains
// connection ownership and must outlive the store.
func NewJSONStore[V any](
	client *Client,
	options StoreOptions,
	observers ...spicecache.Observer,
) (*JSONStore[V], error) {
	if client == nil || client.native == nil || client.cacheCommands == nil {
		return nil, errors.New("construct Redis cache: client is nil")
	}
	return newJSONStore[V](client.cacheCommands, options, observers...)
}

func newJSONStore[V any](
	commands cacheCommands,
	options StoreOptions,
	observers ...spicecache.Observer,
) (*JSONStore[V], error) {
	if commands == nil {
		return nil, errors.New("construct Redis cache: command client is nil")
	}
	if err := normalizeStoreOptions(&options); err != nil {
		return nil, err
	}
	for index, observer := range observers {
		if observer == nil {
			return nil, fmt.Errorf(
				"construct Redis cache %q: observer %d is nil",
				options.Definition.ID,
				index,
			)
		}
	}
	return &JSONStore[V]{
		commands:      commands,
		definition:    options.Definition,
		prefix:        options.Prefix + ":",
		maxValueBytes: options.MaxValueBytes,
		observers:     append([]spicecache.Observer(nil), observers...),
	}, nil
}

// Get returns one decoded entry. Missing and expired Redis values are misses.
func (store *JSONStore[V]) Get(
	ctx context.Context,
	key string,
) (value V, found bool, err error) {
	if validationErr := store.validateOperation(ctx, key, "get"); validationErr != nil {
		return value, false, validationErr
	}
	started := time.Now()
	payload, err := store.commands.getRange(
		ctx,
		store.prefix+key,
		int64(store.maxValueBytes),
	)
	if err != nil {
		return value, false, fmt.Errorf(
			"get Redis cache %q entry: %w",
			store.definition.ID,
			err,
		)
	}
	if len(payload) == 0 {
		store.recordMiss()
		store.observe(ctx, spicecache.Observation{
			Definition: store.definition,
			Operation:  spicecache.OperationGet,
			Duration:   time.Since(started),
		})
		return value, false, nil
	}
	if len(payload) > store.maxValueBytes {
		return value, false, fmt.Errorf(
			"get Redis cache %q entry: encoded value exceeds limit",
			store.definition.ID,
		)
	}
	if err := json.Unmarshal(payload, &value); err != nil {
		return value, false, fmt.Errorf(
			"decode Redis cache %q entry: invalid JSON",
			store.definition.ID,
		)
	}
	store.recordHit()
	store.observe(ctx, spicecache.Observation{
		Definition: store.definition,
		Operation:  spicecache.OperationGet,
		Duration:   time.Since(started),
		Hit:        true,
	})
	return value, true, nil
}

// Put encodes and stores one value. A zero TTL means no expiration.
func (store *JSONStore[V]) Put(
	ctx context.Context,
	key string,
	value V,
	ttl time.Duration,
) error {
	if err := store.validateOperation(ctx, key, "put"); err != nil {
		return err
	}
	if ttl < 0 {
		return errors.New("put Redis cache entry: TTL must not be negative")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf(
			"encode Redis cache %q entry: value is not valid JSON",
			store.definition.ID,
		)
	}
	if len(payload) > store.maxValueBytes {
		return fmt.Errorf(
			"put Redis cache %q entry: encoded value exceeds limit",
			store.definition.ID,
		)
	}
	started := time.Now()
	if err := store.commands.set(ctx, store.prefix+key, payload, ttl); err != nil {
		return fmt.Errorf(
			"put Redis cache %q entry: %w",
			store.definition.ID,
			err,
		)
	}
	store.recordPut()
	store.observe(ctx, spicecache.Observation{
		Definition: store.definition,
		Operation:  spicecache.OperationPut,
		Duration:   time.Since(started),
	})
	return nil
}

// Delete removes one entry. Deleting a missing entry succeeds.
func (store *JSONStore[V]) Delete(ctx context.Context, key string) error {
	if err := store.validateOperation(ctx, key, "delete"); err != nil {
		return err
	}
	started := time.Now()
	removed, err := store.commands.delete(ctx, store.prefix+key)
	if err != nil {
		return fmt.Errorf(
			"delete Redis cache %q entry: %w",
			store.definition.ID,
			err,
		)
	}
	store.recordDelete(removed)
	observation := spicecache.Observation{
		Definition: store.definition,
		Operation:  spicecache.OperationDelete,
		Duration:   time.Since(started),
	}
	if removed {
		observation.Removed = 1
	}
	store.observe(ctx, observation)
	return nil
}

// Snapshot returns local aggregate operation counts. Size, eviction, and
// expiration cardinality are server-wide concerns and remain zero.
func (store *JSONStore[V]) Snapshot() spicecache.Snapshot {
	if store == nil {
		return spicecache.Snapshot{}
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.stats
}

func (commands nativeCacheCommands) getRange(
	ctx context.Context,
	key string,
	maximum int64,
) ([]byte, error) {
	value, err := commands.client.GetRange(ctx, key, 0, maximum).Result()
	return []byte(value), err
}

func (commands nativeCacheCommands) set(
	ctx context.Context,
	key string,
	value []byte,
	ttl time.Duration,
) error {
	return commands.client.Set(ctx, key, value, ttl).Err()
}

func (commands nativeCacheCommands) delete(
	ctx context.Context,
	key string,
) (bool, error) {
	removed, err := commands.client.Del(ctx, key).Result()
	return removed != 0, err
}

func normalizeStoreOptions(options *StoreOptions) error {
	switch {
	case options.Definition.ID == "":
		return errors.New("construct Redis cache: cache ID is required")
	case options.Definition.Module == "":
		return fmt.Errorf(
			"construct Redis cache %q: module is required",
			options.Definition.ID,
		)
	case !validPrefix(options.Prefix):
		return fmt.Errorf(
			"construct Redis cache %q: prefix is invalid",
			options.Definition.ID,
		)
	case options.MaxValueBytes < 0 ||
		options.MaxValueBytes > maxMaxValueBytes:
		return fmt.Errorf(
			"construct Redis cache %q: maximum value bytes is invalid",
			options.Definition.ID,
		)
	}
	if options.MaxValueBytes == 0 {
		options.MaxValueBytes = defaultMaxValueBytes
	}
	return nil
}

func validPrefix(prefix string) bool {
	if prefix == "" || len(prefix) > maxPrefixBytes {
		return false
	}
	for _, character := range []byte(prefix) {
		if !isKeyCharacter(character) {
			return false
		}
	}
	return true
}

func validKey(prefix, key string) bool {
	if key == "" ||
		len(key) > maxKeyBytes ||
		len(prefix)+len(key) > maxRedisKeyBytes ||
		!utf8.ValidString(key) {
		return false
	}
	for _, character := range []byte(key) {
		if !isKeyCharacter(character) && character != ':' && character != '/' {
			return false
		}
	}
	return true
}

func isKeyCharacter(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		strings.ContainsRune("._-", rune(character))
}

func (store *JSONStore[V]) validateOperation(
	ctx context.Context,
	key string,
	operation string,
) error {
	if ctx == nil {
		return fmt.Errorf("%s Redis cache entry: context is nil", operation)
	}
	if store == nil || store.commands == nil {
		return fmt.Errorf("%s Redis cache entry: cache is nil", operation)
	}
	if cause := context.Cause(ctx); cause != nil {
		return fmt.Errorf("%s Redis cache entry: %w", operation, cause)
	}
	if !validKey(store.prefix, key) {
		return fmt.Errorf("%s Redis cache entry: key is invalid", operation)
	}
	return nil
}

func (store *JSONStore[V]) recordHit() {
	store.mu.Lock()
	store.stats.Hits++
	store.mu.Unlock()
}

func (store *JSONStore[V]) recordMiss() {
	store.mu.Lock()
	store.stats.Misses++
	store.mu.Unlock()
}

func (store *JSONStore[V]) recordPut() {
	store.mu.Lock()
	store.stats.Puts++
	store.mu.Unlock()
}

func (store *JSONStore[V]) recordDelete(removed bool) {
	if !removed {
		return
	}
	store.mu.Lock()
	store.stats.Deletes++
	store.mu.Unlock()
}

func (store *JSONStore[V]) observe(
	ctx context.Context,
	observation spicecache.Observation,
) {
	for _, observer := range store.observers {
		observer(ctx, observation)
	}
}

var _ spicecache.Store[string, any] = (*JSONStore[any])(nil)
