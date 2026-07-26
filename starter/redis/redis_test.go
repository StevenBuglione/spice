package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOpenAppliesSecureDeterministicDefaultsWithoutConnecting(t *testing.T) {
	t.Parallel()

	client, cleanup, err := Open(Options{
		URL: "rediss://default:secret@cache.example.test:6380/2",
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := cleanup(context.Background()); closeErr != nil {
			t.Errorf("cleanup() error = %v", closeErr)
		}
	})

	options := client.native.Options()
	if options.Addr != "cache.example.test:6380" ||
		options.DB != 2 ||
		options.Username != "default" ||
		options.Password != "secret" ||
		options.ClientName != defaultClientName ||
		options.Protocol != 2 ||
		options.MaxRetries != defaultMaxRetries ||
		options.DialTimeout != defaultDialTimeout ||
		options.ReadTimeout != defaultReadTimeout ||
		options.WriteTimeout != defaultWriteTimeout ||
		!options.ContextTimeoutEnabled ||
		!options.PoolFIFO ||
		options.PoolSize != defaultPoolSize ||
		options.MaxConcurrentDials != defaultPoolSize ||
		options.MaxActiveConns != defaultPoolSize ||
		options.MinIdleConns != defaultMinimumIdleConnections ||
		options.MaxIdleConns != defaultPoolSize ||
		options.ConnMaxIdleTime != defaultConnectionMaxIdleTime ||
		options.ConnMaxLifetime != defaultConnectionMaxLifetime {
		t.Fatal("native Redis defaults were not applied")
	}
	if options.TLSConfig == nil ||
		options.TLSConfig.MinVersion != tls.VersionTLS12 ||
		options.TLSConfig.ServerName != "cache.example.test" {
		t.Fatalf("TLS options = %#v", options.TLSConfig)
	}
}

func TestOpenPreservesExplicitBoundedOptions(t *testing.T) {
	t.Parallel()

	client, cleanup, err := Open(Options{
		URL:                    "redis://127.0.0.1:6379/0",
		ClientName:             "orders-cache",
		PoolSize:               8,
		MinimumIdleConnections: 3,
		MaxRetries:             4,
		DialTimeout:            time.Second,
		ReadTimeout:            2 * time.Second,
		WriteTimeout:           3 * time.Second,
		ConnectionMaxIdleTime:  4 * time.Minute,
		ConnectionMaxLifetime:  20 * time.Minute,
		AllowInsecure:          true,
		AllowUnauthenticated:   true,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := cleanup(context.Background()); closeErr != nil {
			t.Errorf("cleanup() error = %v", closeErr)
		}
	})
	options := client.native.Options()
	if options.ClientName != "orders-cache" ||
		options.PoolSize != 8 ||
		options.MinIdleConns != 3 ||
		options.MaxRetries != 4 ||
		options.DialTimeout != time.Second ||
		options.ReadTimeout != 2*time.Second ||
		options.WriteTimeout != 3*time.Second ||
		options.ConnMaxIdleTime != 4*time.Minute ||
		options.ConnMaxLifetime != 20*time.Minute ||
		options.TLSConfig != nil {
		t.Fatal("explicit native Redis options changed")
	}
}

func TestOpenRejectsUnsafeOrInvalidConfigurationWithoutExposingSecrets(t *testing.T) {
	t.Parallel()

	valid := Options{
		URL: "rediss://default:super-secret@cache.example.test:6380/0",
	}
	tests := []struct {
		name   string
		mutate func(*Options)
	}{
		{name: "empty URL", mutate: func(options *Options) { options.URL = "" }},
		{name: "large URL", mutate: func(options *Options) {
			options.URL = "rediss://" + strings.Repeat("x", maxConnectionURLBytes)
		}},
		{name: "scheme", mutate: func(options *Options) {
			options.URL = "http://default:secret@cache.example.test:6380/0"
		}},
		{name: "insecure", mutate: func(options *Options) {
			options.URL = "redis://default:secret@cache.example.test:6379/0"
		}},
		{name: "host", mutate: func(options *Options) {
			options.URL = "rediss://default:secret@/0"
		}},
		{name: "port", mutate: func(options *Options) {
			options.URL = "rediss://default:secret@cache.example.test:0/0"
		}},
		{name: "database", mutate: func(options *Options) {
			options.URL = "rediss://default:secret@cache.example.test:6380/one"
		}},
		{name: "nested path", mutate: func(options *Options) {
			options.URL = "rediss://default:secret@cache.example.test:6380/0/1"
		}},
		{name: "query", mutate: func(options *Options) {
			options.URL += "?read_timeout=-1"
		}},
		{name: "fragment", mutate: func(options *Options) {
			options.URL += "#fragment"
		}},
		{name: "authentication", mutate: func(options *Options) {
			options.URL = "rediss://cache.example.test:6380/0"
		}},
		{name: "client name control", mutate: func(options *Options) {
			options.ClientName = "orders\ncache"
		}},
		{name: "client name length", mutate: func(options *Options) {
			options.ClientName = strings.Repeat("x", maxClientNameBytes+1)
		}},
		{name: "pool negative", mutate: func(options *Options) {
			options.PoolSize = -1
		}},
		{name: "pool excess", mutate: func(options *Options) {
			options.PoolSize = maxConfiguredConnectionCount + 1
		}},
		{name: "idle negative", mutate: func(options *Options) {
			options.MinimumIdleConnections = -1
		}},
		{name: "idle excess", mutate: func(options *Options) {
			options.PoolSize = 2
			options.MinimumIdleConnections = 3
		}},
		{name: "retries negative", mutate: func(options *Options) {
			options.MaxRetries = -1
		}},
		{name: "retries excess", mutate: func(options *Options) {
			options.MaxRetries = maxConfiguredRetries + 1
		}},
		{name: "dial timeout", mutate: func(options *Options) {
			options.DialTimeout = -time.Second
		}},
		{name: "read timeout", mutate: func(options *Options) {
			options.ReadTimeout = maxConfiguredOperationTimeout + time.Second
		}},
		{name: "write timeout", mutate: func(options *Options) {
			options.WriteTimeout = -time.Second
		}},
		{name: "idle lifetime", mutate: func(options *Options) {
			options.ConnectionMaxIdleTime = -time.Second
		}},
		{name: "connection lifetime", mutate: func(options *Options) {
			options.ConnectionMaxLifetime =
				maxConfiguredConnectionLifetime + time.Second
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			options := valid
			test.mutate(&options)
			_, _, err := Open(options)
			if err == nil {
				t.Fatal("Open() unexpectedly succeeded")
			}
			if strings.Contains(err.Error(), "super-secret") {
				t.Fatalf("Open() exposed connection secret: %v", err)
			}
		})
	}
}

func TestOpenAllowsExplicitLocalExceptions(t *testing.T) {
	t.Parallel()

	client, cleanup, err := Open(Options{
		URL:                  "redis://127.0.0.1:6379/0",
		PoolSize:             1,
		AllowInsecure:        true,
		AllowUnauthenticated: true,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if client.native.Options().MinIdleConns != 1 {
		t.Fatalf(
			"minimum idle connections = %d, want 1",
			client.native.Options().MinIdleConns,
		)
	}
	if err := cleanup(context.Background()); err != nil {
		t.Fatalf("cleanup() error = %v", err)
	}
	if err := cleanup(context.Background()); err != nil {
		t.Fatalf("second cleanup() error = %v", err)
	}
}

func TestPingAndCloseValidateInputsAndPreserveCancellation(t *testing.T) {
	t.Parallel()

	var nilClient *Client
	if err := nilClient.Ping(context.Background()); err == nil {
		t.Fatal("nil Ping() unexpectedly succeeded")
	}
	if err := nilClient.Close(context.Background()); err == nil {
		t.Fatal("nil Close() unexpectedly succeeded")
	}

	client, cleanup, err := Open(Options{
		URL:                  "redis://127.0.0.1:1/0",
		AllowInsecure:        true,
		AllowUnauthenticated: true,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := cleanup(context.Background()); closeErr != nil {
			t.Errorf("cleanup() error = %v", closeErr)
		}
	})
	if err := client.Ping(nilTestContext()); err == nil {
		t.Fatal("Ping(nil) unexpectedly succeeded")
	}
	if err := client.Close(nilTestContext()); err == nil {
		t.Fatal("Close(nil) unexpectedly succeeded")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Ping(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Ping(canceled) error = %v, want context.Canceled", err)
	}
}

func nilTestContext() context.Context {
	return nil
}
