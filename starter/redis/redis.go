// Package redis provides a reviewed go-redis-backed client starter for Spice
// applications.
package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	spicelifecycle "github.com/StevenBuglione/spice/lifecycle"
	redisclient "github.com/redis/go-redis/v9"
)

const (
	defaultClientName               = "spice"
	defaultPoolSize                 = 20
	defaultMinimumIdleConnections   = 2
	defaultMaxRetries               = 2
	defaultDialTimeout              = 5 * time.Second
	defaultReadTimeout              = 3 * time.Second
	defaultWriteTimeout             = 3 * time.Second
	defaultConnectionMaxIdleTime    = 5 * time.Minute
	defaultConnectionMaxLifetime    = 30 * time.Minute
	maxConnectionURLBytes           = 16 << 10
	maxClientNameBytes              = 128
	maxConfiguredConnectionCount    = 100_000
	maxConfiguredRetries            = 10
	maxConfiguredOperationTimeout   = 5 * time.Minute
	maxConfiguredConnectionLifetime = 24 * time.Hour
)

// Options defines one standalone Redis client. URL must be a complete redis or
// rediss URL. TLS and authenticated access are secure defaults; local
// development must opt into either exception explicitly.
type Options struct {
	URL                    string
	ClientName             string
	PoolSize               int
	MinimumIdleConnections int
	MaxRetries             int
	DialTimeout            time.Duration
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	ConnectionMaxIdleTime  time.Duration
	ConnectionMaxLifetime  time.Duration
	AllowInsecure          bool
	AllowUnauthenticated   bool
}

// Client owns one standalone Redis connection pool.
type Client struct {
	native   *redisclient.Client
	close    sync.Once
	closeErr error
}

// Open validates deterministic connection policy and constructs a
// caller-owned Redis client. It performs no network I/O.
func Open(options Options) (*Client, spicelifecycle.Cleanup, error) {
	nativeOptions, err := nativeOptions(options)
	if err != nil {
		return nil, nil, err
	}
	client := &Client{native: redisclient.NewClient(nativeOptions)}
	cleanup := spicelifecycle.Cleanup(client.Close)
	return client, cleanup, nil
}

// Ping verifies one client with the caller-owned context.
func (client *Client) Ping(ctx context.Context) error {
	switch {
	case ctx == nil:
		return errors.New("ping Redis client: context is nil")
	case client == nil || client.native == nil:
		return errors.New("ping Redis client: client is nil")
	}
	if err := client.native.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping Redis client: %w", err)
	}
	return nil
}

// Close releases the connection pool once. The context is retained for the
// lifecycle cleanup contract; closing go-redis itself is synchronous.
func (client *Client) Close(ctx context.Context) error {
	switch {
	case ctx == nil:
		return errors.New("close Redis client: context is nil")
	case client == nil || client.native == nil:
		return errors.New("close Redis client: client is nil")
	}
	client.close.Do(func() {
		client.closeErr = client.native.Close()
	})
	if client.closeErr != nil {
		return fmt.Errorf("close Redis client: %w", client.closeErr)
	}
	return nil
}

func nativeOptions(options Options) (*redisclient.Options, error) {
	parsed, err := validateURL(options)
	if err != nil {
		return nil, err
	}
	normalizeOptions(&options)
	if validationErr := validateOptions(options); validationErr != nil {
		return nil, validationErr
	}

	native, parseErr := redisclient.ParseURL(options.URL)
	if parseErr != nil || native == nil {
		return nil, errors.New("construct Redis client: connection URL is invalid")
	}
	native.ClientName = options.ClientName
	native.Protocol = 2
	native.MaxRetries = options.MaxRetries
	native.DialTimeout = options.DialTimeout
	native.ReadTimeout = options.ReadTimeout
	native.WriteTimeout = options.WriteTimeout
	native.ContextTimeoutEnabled = true
	native.PoolFIFO = true
	native.PoolSize = options.PoolSize
	native.MaxConcurrentDials = options.PoolSize
	native.MaxActiveConns = options.PoolSize
	native.MinIdleConns = options.MinimumIdleConnections
	native.MaxIdleConns = options.PoolSize
	native.PoolTimeout = options.ReadTimeout + time.Second
	native.ConnMaxIdleTime = options.ConnectionMaxIdleTime
	native.ConnMaxLifetime = options.ConnectionMaxLifetime
	if parsed.Scheme == "rediss" {
		native.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: parsed.Hostname(),
		}
	}
	return native, nil
}

func validateURL(options Options) (*url.URL, error) {
	if options.URL == "" || len(options.URL) > maxConnectionURLBytes {
		return nil, errors.New("construct Redis client: connection URL is required")
	}
	parsed, err := url.Parse(options.URL)
	if err != nil || parsed == nil || parsed.Opaque != "" {
		return nil, errors.New("construct Redis client: connection URL is invalid")
	}
	if err := validateTransport(parsed.Scheme, options.AllowInsecure); err != nil {
		return nil, err
	}
	if !validNetworkURL(parsed) {
		return nil, errors.New("construct Redis client: connection URL is incomplete")
	}
	if !authenticated(parsed) && !options.AllowUnauthenticated {
		return nil, errors.New("construct Redis client: authenticated access is required")
	}
	return parsed, nil
}

func validateTransport(scheme string, allowInsecure bool) error {
	switch scheme {
	case "rediss":
		return nil
	case "redis":
		if !allowInsecure {
			return errors.New("construct Redis client: insecure transport is not permitted")
		}
		return nil
	default:
		return errors.New("construct Redis client: connection URL scheme is not permitted")
	}
}

func validNetworkURL(parsed *url.URL) bool {
	return parsed.Host != "" &&
		parsed.Hostname() != "" &&
		parsed.Fragment == "" &&
		parsed.RawQuery == "" &&
		validPort(parsed.Port()) &&
		validDatabasePath(parsed.Path)
}

func authenticated(parsed *url.URL) bool {
	if parsed.User == nil {
		return false
	}
	password, present := parsed.User.Password()
	return present && password != ""
}

func validPort(port string) bool {
	if port == "" {
		return true
	}
	value, err := strconv.ParseUint(port, 10, 16)
	return err == nil && value > 0
}

func validDatabasePath(path string) bool {
	if path == "" || path == "/" {
		return true
	}
	if !strings.HasPrefix(path, "/") ||
		strings.Contains(strings.TrimPrefix(path, "/"), "/") {
		return false
	}
	database, err := strconv.ParseUint(strings.TrimPrefix(path, "/"), 10, 31)
	return err == nil && database <= uint64(^uint(0)>>1)
}

func normalizeOptions(options *Options) {
	if options.ClientName == "" {
		options.ClientName = defaultClientName
	}
	if options.PoolSize == 0 {
		options.PoolSize = defaultPoolSize
	}
	if options.MinimumIdleConnections == 0 {
		options.MinimumIdleConnections = min(
			defaultMinimumIdleConnections,
			options.PoolSize,
		)
	}
	if options.MaxRetries == 0 {
		options.MaxRetries = defaultMaxRetries
	}
	if options.DialTimeout == 0 {
		options.DialTimeout = defaultDialTimeout
	}
	if options.ReadTimeout == 0 {
		options.ReadTimeout = defaultReadTimeout
	}
	if options.WriteTimeout == 0 {
		options.WriteTimeout = defaultWriteTimeout
	}
	if options.ConnectionMaxIdleTime == 0 {
		options.ConnectionMaxIdleTime = defaultConnectionMaxIdleTime
	}
	if options.ConnectionMaxLifetime == 0 {
		options.ConnectionMaxLifetime = defaultConnectionMaxLifetime
	}
}

func validateOptions(options Options) error {
	if !validClientName(options.ClientName) {
		return errors.New("construct Redis client: client name is invalid")
	}
	if err := validatePoolOptions(options); err != nil {
		return err
	}
	return validateTimeoutOptions(options)
}

func validatePoolOptions(options Options) error {
	switch {
	case options.PoolSize < 1 ||
		options.PoolSize > maxConfiguredConnectionCount:
		return errors.New("construct Redis client: pool size is invalid")
	case options.MinimumIdleConnections < 0 ||
		options.MinimumIdleConnections > options.PoolSize:
		return errors.New("construct Redis client: minimum idle connections is invalid")
	case options.MaxRetries < 1 || options.MaxRetries > maxConfiguredRetries:
		return errors.New("construct Redis client: max retries is invalid")
	default:
		return nil
	}
}

func validateTimeoutOptions(options Options) error {
	switch {
	case !validOperationTimeout(options.DialTimeout):
		return errors.New("construct Redis client: dial timeout is invalid")
	case !validOperationTimeout(options.ReadTimeout):
		return errors.New("construct Redis client: read timeout is invalid")
	case !validOperationTimeout(options.WriteTimeout):
		return errors.New("construct Redis client: write timeout is invalid")
	case options.ConnectionMaxIdleTime < 0 ||
		options.ConnectionMaxIdleTime > maxConfiguredConnectionLifetime:
		return errors.New("construct Redis client: connection max idle time is invalid")
	case options.ConnectionMaxLifetime < 0 ||
		options.ConnectionMaxLifetime > maxConfiguredConnectionLifetime:
		return errors.New("construct Redis client: connection max lifetime is invalid")
	default:
		return nil
	}
}

func validClientName(name string) bool {
	if name == "" || len(name) > maxClientNameBytes {
		return false
	}
	for _, character := range []byte(name) {
		if character < 0x21 ||
			character > 0x7e ||
			character == ' ' {
			return false
		}
	}
	return true
}

func validOperationTimeout(value time.Duration) bool {
	return value > 0 && value <= maxConfiguredOperationTimeout
}
