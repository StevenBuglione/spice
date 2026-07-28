package mysql

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOpenConfiguresSecurePoolWithoutConnecting(t *testing.T) {
	t.Parallel()

	database, err := Open(Options{
		URL: "mysql://spice:secret@127.0.0.1:1/spice",
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := database.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})
	if database.Stats().MaxOpenConnections != defaultMaxOpenConnections {
		t.Fatalf("MaxOpenConnections = %d", database.Stats().MaxOpenConnections)
	}
	config, normalized, err := driverConfig(Options{
		URL: "mysql://spice:secret@database.example.test:3306/application",
	})
	if err != nil {
		t.Fatalf("driverConfig() error = %v", err)
	}
	if config.TLS == nil ||
		config.TLS.ServerName != "database.example.test" ||
		config.TLS.MinVersion != tls.VersionTLS12 ||
		!config.ParseTime ||
		!config.ClientFoundRows ||
		config.MultiStatements ||
		config.AllowAllFiles ||
		config.ConnectionAttributes != "program_name:spice" {
		t.Fatalf("driver config = %#v", config)
	}
	if normalized.ConnectionMaxLifetime != defaultConnectionMaxLifetime ||
		normalized.ConnectionMaxIdleTime != defaultConnectionMaxIdleTime ||
		normalized.ConnectTimeout != defaultConnectTimeout {
		t.Fatalf("normalized options = %#v", normalized)
	}
}

func TestOpenAllowsOnlyExplicitLocalInsecurity(t *testing.T) {
	t.Parallel()

	config, _, err := driverConfig(Options{
		URL:           "mysql://spice:secret@127.0.0.1:3306/spice?tls=disable",
		AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("driverConfig() error = %v", err)
	}
	if config.TLS != nil {
		t.Fatal("insecure opt-in retained a TLS config")
	}
}

func TestOpenRejectsInvalidOptionsWithoutExposingSecrets(t *testing.T) {
	t.Parallel()

	valid := Options{
		URL: "mysql://spice:super-secret@database.example.test:3306/application",
	}
	tests := []struct {
		name   string
		mutate func(*Options)
	}{
		{name: "empty URL", mutate: func(options *Options) { options.URL = "" }},
		{name: "scheme", mutate: func(options *Options) {
			options.URL = "postgres://spice:secret@database.example.test:3306/application"
		}},
		{name: "host", mutate: func(options *Options) {
			options.URL = "mysql://spice:secret@:3306/application"
		}},
		{name: "port", mutate: func(options *Options) {
			options.URL = "mysql://spice:secret@database.example.test/application"
		}},
		{name: "user", mutate: func(options *Options) {
			options.URL = "mysql://:secret@database.example.test:3306/application"
		}},
		{name: "password", mutate: func(options *Options) {
			options.URL = "mysql://spice@database.example.test:3306/application"
		}},
		{name: "database", mutate: func(options *Options) {
			options.URL = "mysql://spice:secret@database.example.test:3306/"
		}},
		{name: "unknown option", mutate: func(options *Options) {
			options.URL += "?multiStatements=true"
		}},
		{name: "implicit insecurity", mutate: func(options *Options) {
			options.URL += "?tls=disable"
		}},
		{name: "application control", mutate: func(options *Options) {
			options.ApplicationName = "orders\nservice"
		}},
		{name: "application delimiter", mutate: func(options *Options) {
			options.ApplicationName = "orders:service"
		}},
		{name: "max open", mutate: func(options *Options) {
			options.MaxOpenConnections = -1
		}},
		{name: "max idle", mutate: func(options *Options) {
			options.MaxOpenConnections = 2
			options.MaxIdleConnections = 3
		}},
		{name: "lifetime", mutate: func(options *Options) {
			options.ConnectionMaxLifetime = -time.Second
		}},
		{name: "connect timeout", mutate: func(options *Options) {
			options.ConnectTimeout = -time.Second
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			options := valid
			test.mutate(&options)
			_, err := Open(options)
			if err == nil {
				t.Fatal("Open() unexpectedly succeeded")
			}
			if strings.Contains(err.Error(), "super-secret") {
				t.Fatalf("error exposed secret: %v", err)
			}
		})
	}
}

func TestPingValidatesInputsAndCancellation(t *testing.T) {
	t.Parallel()

	if err := Ping(nilTestContext(), nil); err == nil ||
		!strings.Contains(err.Error(), "context is nil") {
		t.Fatalf("Ping(nil) error = %v", err)
	}
	if err := Ping(context.Background(), nil); err == nil ||
		!strings.Contains(err.Error(), "database is nil") {
		t.Fatalf("Ping(nil database) error = %v", err)
	}
	database, err := Open(Options{
		URL:           "mysql://spice:secret@127.0.0.1:1/spice?tls=disable",
		AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := database.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Ping(ctx, database); !errors.Is(err, context.Canceled) {
		t.Fatalf("Ping() error = %v", err)
	}
}

func nilTestContext() context.Context {
	return nil
}
