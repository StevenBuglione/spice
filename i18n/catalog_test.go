package i18n

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestCatalogNegotiatesAndFallsBackDeterministically(t *testing.T) {
	t.Parallel()

	catalog, err := ParseProperties(fstest.MapFS{
		"messages.properties":    {Data: []byte("welcome=Welcome\nowner=Owner\n")},
		"messages_de.properties": {Data: []byte("welcome=Willkommen\n")},
		"messages_es.properties": {Data: []byte("welcome=Bienvenido\n")},
	}, "messages*.properties", "en")
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.Locales(); strings.Join(got, ",") != "de,en,es" {
		t.Fatalf("Locales() = %q", got)
	}
	tests := []struct {
		header string
		want   string
	}{
		{"de-DE,de;q=0.9,en;q=0.8", "de"},
		{"fr;q=0.8,es;q=0.9", "es"},
		{"*", "en"},
		{"malformed;q=wrong", "en"},
		{strings.Repeat("x", maxLanguageBytes+1), "en"},
	}
	for _, test := range tests {
		if got := catalog.Resolve(test.header); got != test.want {
			t.Errorf("Resolve(%q) = %q, want %q", test.header, got, test.want)
		}
	}
	if got, err := catalog.Message("de", "welcome"); err != nil || got != "Willkommen" {
		t.Fatalf("Message(de, welcome) = %q, %v", got, err)
	}
	if got, err := catalog.Message("de", "owner"); err != nil || got != "Owner" {
		t.Fatalf("Message(de, owner) fallback = %q, %v", got, err)
	}
	if _, err := catalog.Message("de", "missing"); err == nil {
		t.Fatal("Message(de, missing) error = nil")
	}
}

func TestCatalogRejectsInvalidSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  fstest.MapFS
		pattern string
		def     string
		want    string
	}{
		{"missing", fstest.MapFS{}, "*.properties", "en", "between 1"},
		{"missing default", fstest.MapFS{"messages_de.properties": {Data: []byte("a=b")}}, "*.properties", "en", "has no catalog"},
		{"duplicate key", fstest.MapFS{"messages.properties": {Data: []byte("a=b\na=c")}}, "*.properties", "en", "duplicated"},
		{"bad line", fstest.MapFS{"messages.properties": {Data: []byte("missing")}}, "*.properties", "en", "key=value"},
		{"bad escape", fstest.MapFS{"messages.properties": {Data: []byte(`a=\u0041`)}}, "*.properties", "en", "unsupported escape"},
		{"bad locale", fstest.MapFS{"messages_bad!.properties": {Data: []byte("a=b")}}, "*.properties", "en", "locale"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseProperties(test.source, test.pattern, test.def)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParseProperties() error = %v, want %q", err, test.want)
			}
		})
	}
	if _, err := ParseProperties(nil, "*.properties", "en"); err == nil {
		t.Fatal("ParseProperties(nil) error = nil")
	}
}
