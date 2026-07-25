package config

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestResolveAppliesDeterministicPrecedenceProvenanceAndRedaction(t *testing.T) {
	t.Parallel()
	schema := MustSchema(
		Property{
			Key:         "database.password",
			Kind:        KindString,
			Environment: "SHOP_DATABASE_PASSWORD",
			Required:    true,
			Secret:      true,
			Module:      "example.com/shop/orders",
		},
		Property{
			Key:        "feature.enabled",
			Kind:       KindBoolean,
			Default:    "false",
			HasDefault: true,
		},
		Property{
			Key:        "server.port",
			Kind:       KindInteger,
			Default:    "8080",
			HasDefault: true,
		},
		Property{
			Key:        "server.timeout",
			Kind:       KindDuration,
			Default:    "5s",
			HasDefault: true,
		},
	)
	values := map[string]string{
		"database.password": "map-secret",
		"feature.enabled":   "true",
	}
	overrides, err := NewMapSource("overrides", values)
	if err != nil {
		t.Fatalf("NewMapSource() error = %v", err)
	}
	values["feature.enabled"] = "false"
	environment, err := NewEnvironmentSource(
		"environment",
		"SHOP_",
		func(name string) (string, bool) {
			environmentValues := map[string]string{
				"SHOP_DATABASE_PASSWORD": "environment-secret",
				"SHOP_SERVER_PORT":       "9090",
			}
			value, ok := environmentValues[name]
			return value, ok
		},
	)
	if err != nil {
		t.Fatalf("NewEnvironmentSource() error = %v", err)
	}
	profileProbe := &recordingSource{name: "profiles"}

	snapshot, err := Resolve(
		context.Background(),
		schema,
		Options{Profiles: []string{"dev", "local"}},
		overrides,
		profileProbe,
		environment,
	)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !slices.Equal(profileProbe.profiles, []string{"dev", "local"}) {
		t.Fatalf("source profiles = %v", profileProbe.profiles)
	}
	if got, want := snapshot.Keys(), []string{
		"database.password",
		"feature.enabled",
		"server.port",
		"server.timeout",
	}; !slices.Equal(got, want) {
		t.Fatalf("Keys() = %v, want %v", got, want)
	}
	password, passwordErr := snapshot.RequiredString("database.password")
	if passwordErr != nil || password != "environment-secret" {
		t.Fatalf("password = %q", password)
	}
	enabled, enabledErr := snapshot.Boolean("feature.enabled")
	if enabledErr != nil || !enabled {
		t.Fatalf("Boolean(feature.enabled) = %t, %v", enabled, enabledErr)
	}
	port, portErr := snapshot.Integer("server.port")
	if portErr != nil || port != 9090 {
		t.Fatalf("Integer(server.port) = %d, %v", port, portErr)
	}
	timeout, timeoutErr := snapshot.Duration("server.timeout")
	if timeoutErr != nil || timeout != 5*time.Second {
		t.Fatalf("Duration(server.timeout) = %v, %v", timeout, timeoutErr)
	}
	passwordEntry, ok := snapshot.Entry("database.password")
	if !ok ||
		passwordEntry.Origin.Source != "environment" ||
		passwordEntry.Origin.Default ||
		!passwordEntry.Secret {
		t.Fatalf("password entry = %#v, %t", passwordEntry, ok)
	}
	timeoutEntry, ok := snapshot.Entry("server.timeout")
	if !ok || timeoutEntry.Origin.Source != "default" || !timeoutEntry.Origin.Default {
		t.Fatalf("timeout entry = %#v, %t", timeoutEntry, ok)
	}
	redacted := snapshot.Redacted()
	if redacted["database.password"] != redactedValue ||
		strings.Contains(snapshot.String(), "environment-secret") ||
		!strings.Contains(snapshot.String(), "database.password="+redactedValue) {
		t.Fatalf("redacted snapshot = %v\n%s", redacted, snapshot.String())
	}

	properties := schema.Properties()
	properties[0].Key = "changed"
	if schema.Properties()[0].Key == "changed" {
		t.Fatal("Properties returned mutable storage")
	}
	loaded, err := overrides.Load(context.Background(), Request{})
	if err != nil {
		t.Fatalf("MapSource.Load() error = %v", err)
	}
	loaded["feature.enabled"] = "changed"
	reloaded, err := overrides.Load(context.Background(), Request{})
	if err != nil || reloaded["feature.enabled"] != "true" {
		t.Fatalf("MapSource was mutated: %v, %v", reloaded, err)
	}
}

func TestNewSchemaRejectsInvalidMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		properties []Property
		expected   string
	}{
		{
			name:       "key",
			properties: []Property{{Key: "Server.Port", Kind: KindInteger}},
			expected:   "must match",
		},
		{
			name:       "kind",
			properties: []Property{{Key: "server.port", Kind: Kind("float")}},
			expected:   "unsupported kind",
		},
		{
			name: "environment",
			properties: []Property{{
				Key: "server.port", Kind: KindInteger, Environment: "server_port",
			}},
			expected: "environment variable",
		},
		{
			name: "default",
			properties: []Property{{
				Key: "server.port", Kind: KindInteger, Default: "secret-invalid", HasDefault: true,
			}},
			expected: "default is invalid",
		},
		{
			name: "duplicate-key",
			properties: []Property{
				{Key: "server.port", Kind: KindInteger},
				{Key: "server.port", Kind: KindInteger},
			},
			expected: "duplicate configuration property",
		},
		{
			name: "duplicate-environment",
			properties: []Property{
				{Key: "server.port", Kind: KindInteger, Environment: "PORT"},
				{Key: "admin.port", Kind: KindInteger, Environment: "PORT"},
			},
			expected: "same environment variable",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewSchema(test.properties...); err == nil ||
				!strings.Contains(err.Error(), test.expected) {
				t.Fatalf("NewSchema() error = %v, want %q", err, test.expected)
			}
		})
	}
	assertPanics(t, func() {
		MustSchema(Property{Key: "invalid key", Kind: KindString})
	})
}

func TestResolveRejectsInvalidSourcesProfilesAndValues(t *testing.T) {
	t.Parallel()
	requiredSchema := MustSchema(Property{
		Key: "server.port", Kind: KindInteger, Required: true,
	})
	valid, err := NewMapSource("valid", map[string]string{"server.port": "8080"})
	if err != nil {
		t.Fatal(err)
	}
	invalidScalar, err := NewMapSource("invalid", map[string]string{"server.port": "not-a-port"})
	if err != nil {
		t.Fatal(err)
	}
	unknown, err := NewMapSource("unknown", map[string]string{"server.host": "localhost"})
	if err != nil {
		t.Fatal(err)
	}
	failing := stubSource{name: "failing", err: errors.New("load failed")}
	invalidName := stubSource{name: " invalid"}
	//nolint:staticcheck // Explicitly verifies the defensive nil-context contract.
	if _, err := Resolve(nil, requiredSchema, Options{}); err == nil ||
		!strings.Contains(err.Error(), "context is nil") {
		t.Fatalf("Resolve(nil context) error = %v", err)
	}
	tests := []struct {
		name     string
		options  Options
		sources  []Source
		expected string
	}{
		{
			name:    "invalid-profile",
			options: Options{Profiles: []string{"Bad"}}, expected: "must match",
		},
		{
			name:    "duplicate-profile",
			options: Options{Profiles: []string{"dev", "dev"}}, expected: "active more than once",
		},
		{
			name:    "nil-source",
			sources: []Source{nil}, expected: "source 0 is nil",
		},
		{
			name:    "invalid-source-name",
			sources: []Source{invalidName}, expected: "invalid name",
		},
		{
			name:    "duplicate-source",
			sources: []Source{valid, valid}, expected: "duplicate source name",
		},
		{
			name:    "source-failure",
			sources: []Source{failing}, expected: "load failed",
		},
		{
			name:    "unknown",
			sources: []Source{unknown}, expected: "unknown property",
		},
		{
			name:     "required",
			expected: "required configuration property",
		},
		{
			name:    "scalar",
			sources: []Source{invalidScalar}, expected: "base-10 signed 64-bit integer",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, resolveErr := Resolve(
				context.Background(),
				requiredSchema,
				test.options,
				test.sources...,
			)
			if resolveErr == nil || !strings.Contains(resolveErr.Error(), test.expected) {
				t.Fatalf("Resolve() error = %v, want %q", resolveErr, test.expected)
			}
		})
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Resolve(canceled, requiredSchema, Options{}, valid); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("Resolve(canceled) error = %v", err)
	}
	if snapshot, err := Resolve(
		context.Background(),
		requiredSchema,
		Options{AllowUnknown: true},
		valid,
		unknown,
	); err != nil || len(snapshot.Keys()) != 2 {
		t.Fatalf("Resolve(allow unknown) = %#v, %v", snapshot, err)
	}
}

func TestEnvironmentSourceUsesDeclaredKeysAndRejectsCollisions(t *testing.T) {
	t.Parallel()
	if _, err := NewEnvironmentSource("", "APP_", func(string) (string, bool) {
		return "", false
	}); err == nil {
		t.Fatal("NewEnvironmentSource(empty name) error = nil")
	}
	if _, err := NewEnvironmentSource("environment", "bad-", func(string) (string, bool) {
		return "", false
	}); err == nil {
		t.Fatal("NewEnvironmentSource(invalid prefix) error = nil")
	}
	if _, err := NewEnvironmentSource("environment", "APP_", nil); err == nil {
		t.Fatal("NewEnvironmentSource(nil lookup) error = nil")
	}
	if source, err := OSEnvironment("SPICE_"); err != nil || source.Name() != "environment" {
		t.Fatalf("OSEnvironment() = %#v, %v", source, err)
	}

	schema := MustSchema(
		Property{Key: "server-port", Kind: KindInteger},
		Property{Key: "server.port", Kind: KindInteger},
	)
	source, err := NewEnvironmentSource(
		"environment",
		"",
		func(string) (string, bool) { return "8080", true },
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(context.Background(), schema, Options{}, source); err == nil ||
		!strings.Contains(err.Error(), "map to environment variable") {
		t.Fatalf("Resolve(environment collision) error = %v", err)
	}
}

func TestSnapshotScalarAccessorsDoNotLeakValues(t *testing.T) {
	t.Parallel()
	snapshot := newSnapshot(map[string]Entry{
		"boolean":  {Key: "boolean", Value: "secret-bool", Secret: true},
		"duration": {Key: "duration", Value: "secret-duration", Secret: true},
		"integer":  {Key: "integer", Value: "secret-integer", Secret: true},
	})
	for name, invoke := range map[string]func() error{
		"boolean": func() error {
			_, err := snapshot.Boolean("boolean")
			return err
		},
		"duration": func() error {
			_, err := snapshot.Duration("duration")
			return err
		},
		"integer": func() error {
			_, err := snapshot.Integer("integer")
			return err
		},
		"missing": func() error {
			_, err := snapshot.RequiredString("missing")
			return err
		},
	} {
		err := invoke()
		if err == nil || strings.Contains(err.Error(), "secret-") {
			t.Fatalf("%s error = %v", name, err)
		}
	}
	if _, ok := snapshot.Entry("missing"); ok {
		t.Fatal("Entry(missing) found a value")
	}
}

func TestDecodeRunsGeneratedBinderAndValidators(t *testing.T) {
	t.Parallel()
	type serverConfig struct {
		Port int64
	}
	snapshot := newSnapshot(map[string]Entry{
		"server.port": {Key: "server.port", Value: "8080"},
	})
	var validations []int
	decoded, err := Decode(
		context.Background(),
		snapshot,
		func(snapshot Snapshot) (serverConfig, error) {
			port, parseErr := snapshot.Integer("server.port")
			return serverConfig{Port: port}, parseErr
		},
		func(_ context.Context, value serverConfig) error {
			validations = append(validations, 1)
			if value.Port < 1024 {
				return errors.New("privileged port")
			}
			return nil
		},
		func(context.Context, serverConfig) error {
			validations = append(validations, 2)
			return nil
		},
	)
	if err != nil || decoded.Port != 8080 || !slices.Equal(validations, []int{1, 2}) {
		t.Fatalf("Decode() = %#v, %v, validations=%v", decoded, err, validations)
	}

	//nolint:staticcheck // Explicitly verifies the defensive nil-context contract.
	if _, err := Decode[serverConfig](nil, snapshot, nil); err == nil {
		t.Fatal("Decode(nil context) error = nil")
	}
	if _, err := Decode[serverConfig](context.Background(), snapshot, nil); err == nil {
		t.Fatal("Decode(nil decoder) error = nil")
	}
	if _, err := Decode(
		context.Background(),
		snapshot,
		func(Snapshot) (serverConfig, error) { return serverConfig{}, errors.New("bind") },
	); err == nil || !strings.Contains(err.Error(), "bind") {
		t.Fatalf("Decode(binding failure) error = %v", err)
	}
	if _, err := Decode(
		context.Background(),
		snapshot,
		func(Snapshot) (serverConfig, error) { return serverConfig{}, nil },
		Validator[serverConfig](nil),
	); err == nil || !strings.Contains(err.Error(), "validator 0 is nil") {
		t.Fatalf("Decode(nil validator) error = %v", err)
	}
	if _, err := Decode(
		context.Background(),
		snapshot,
		func(Snapshot) (serverConfig, error) { return serverConfig{}, nil },
		func(context.Context, serverConfig) error { return errors.New("invalid") },
	); err == nil || !strings.Contains(err.Error(), "validator 0") {
		t.Fatalf("Decode(validation failure) error = %v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Decode(
		canceled,
		snapshot,
		func(Snapshot) (serverConfig, error) { return serverConfig{}, nil },
		func(context.Context, serverConfig) error { return nil },
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("Decode(canceled validation) error = %v", err)
	}
}

type recordingSource struct {
	name     string
	profiles []string
}

func (source *recordingSource) Name() string {
	return source.name
}

func (source *recordingSource) Load(
	_ context.Context,
	request Request,
) (map[string]string, error) {
	source.profiles = request.Profiles()
	properties := request.Properties()
	if len(properties) != 4 {
		return nil, errors.New("source did not receive schema properties")
	}
	return map[string]string{}, nil
}

type stubSource struct {
	name   string
	values map[string]string
	err    error
}

func (source stubSource) Name() string {
	return source.name
}

func (source stubSource) Load(context.Context, Request) (map[string]string, error) {
	return source.values, source.err
}

func assertPanics(t *testing.T, invoke func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("function did not panic")
		}
	}()
	invoke()
}
