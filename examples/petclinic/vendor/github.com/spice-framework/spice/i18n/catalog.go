// Package i18n provides immutable, bounded message catalogs and deterministic
// HTTP language negotiation.
package i18n

import (
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"path"
	"slices"
	"strconv"
	"strings"
)

const (
	maxCatalogFiles  = 32
	maxCatalogBytes  = 1 << 20
	maxMessageCount  = 4096
	maxLanguageBytes = 4096
	maxLanguageParts = 32
)

// Catalog is an immutable set of locale-keyed messages.
type Catalog struct {
	defaultLocale string
	locales       []string
	messages      map[string]map[string]string
}

type languagePreference struct {
	locale  string
	order   int
	quality float64
}

// ParseProperties parses strict UTF-8 key=value .properties catalogs matched
// by pattern. The base filename is the default catalog; suffixed files such as
// messages_de.properties provide locale overrides.
func ParseProperties(
	source fs.FS,
	pattern string,
	defaultLocale string,
) (*Catalog, error) {
	if source == nil {
		return nil, errors.New("parse message catalog: source filesystem is nil")
	}
	normalizedDefault, err := normalizeLocale(defaultLocale)
	if err != nil {
		return nil, fmt.Errorf("parse message catalog: default locale: %w", err)
	}
	if validationErr := validateCatalogPattern(pattern); validationErr != nil {
		return nil, validationErr
	}
	files, err := fs.Glob(source, pattern)
	if err != nil {
		return nil, fmt.Errorf("parse message catalog: pattern: %w", err)
	}
	if len(files) < 1 || len(files) > maxCatalogFiles {
		return nil, fmt.Errorf(
			"parse message catalog: pattern must match between 1 and %d files",
			maxCatalogFiles,
		)
	}
	slices.Sort(files)
	catalog := &Catalog{
		defaultLocale: normalizedDefault,
		messages:      make(map[string]map[string]string),
	}
	remaining := int64(maxCatalogBytes)
	for _, file := range files {
		locale, values, loadErr := loadCatalogFile(
			source,
			file,
			normalizedDefault,
			&remaining,
		)
		if loadErr != nil {
			return nil, loadErr
		}
		if _, duplicate := catalog.messages[locale]; duplicate {
			return nil, fmt.Errorf(
				"parse message catalog: locale %q is declared more than once",
				locale,
			)
		}
		catalog.messages[locale] = values
		catalog.locales = append(catalog.locales, locale)
	}
	if _, exists := catalog.messages[normalizedDefault]; !exists {
		return nil, fmt.Errorf(
			"parse message catalog: default locale %q has no catalog",
			normalizedDefault,
		)
	}
	slices.Sort(catalog.locales)
	return catalog, nil
}

// Locales returns the supported locale identifiers in lexical order.
func (catalog *Catalog) Locales() []string {
	if catalog == nil {
		return nil
	}
	return slices.Clone(catalog.locales)
}

// Resolve selects a supported locale from an Accept-Language value. Invalid,
// oversized, or unmatched input deterministically falls back to the default.
func (catalog *Catalog) Resolve(header string) string {
	if catalog == nil {
		return ""
	}
	if len(header) > maxLanguageBytes {
		return catalog.defaultLocale
	}
	preferences := parseLanguagePreferences(header)
	for _, candidate := range preferences {
		if matched := catalog.match(candidate.locale); matched != "" {
			return matched
		}
	}
	return catalog.defaultLocale
}

func parseLanguagePreferences(header string) []languagePreference {
	preferences := make([]languagePreference, 0, 4)
	for part := range strings.SplitSeq(header, ",") {
		if len(preferences) == maxLanguageParts {
			return nil
		}
		mediaType, parameters, err := mime.ParseMediaType(
			strings.TrimSpace(part),
		)
		if err != nil {
			continue
		}
		quality := 1.0
		if raw := parameters["q"]; raw != "" {
			quality, err = strconv.ParseFloat(raw, 64)
			if err != nil || quality <= 0 || quality > 1 {
				continue
			}
		}
		locale, err := normalizeLocale(mediaType)
		if err != nil && mediaType != "*" {
			continue
		}
		preferences = append(preferences, languagePreference{
			locale: locale, order: len(preferences), quality: quality,
		})
	}
	slices.SortStableFunc(preferences, func(left, right languagePreference) int {
		switch {
		case left.quality > right.quality:
			return -1
		case left.quality < right.quality:
			return 1
		default:
			return left.order - right.order
		}
	})
	return preferences
}

func (catalog *Catalog) match(locale string) string {
	if locale == "" {
		return catalog.defaultLocale
	}
	if _, exists := catalog.messages[locale]; exists {
		return locale
	}
	if separator := strings.IndexByte(locale, '-'); separator > 0 {
		base := locale[:separator]
		if _, exists := catalog.messages[base]; exists {
			return base
		}
	}
	return ""
}

// Message returns a localized message, falling back to the default catalog
// when the selected locale does not override key.
func (catalog *Catalog) Message(locale, key string) (string, error) {
	if catalog == nil {
		return "", errors.New("resolve message: catalog is nil")
	}
	if key == "" || strings.TrimSpace(key) != key {
		return "", errors.New("resolve message: key must be non-empty and trimmed")
	}
	normalized, err := normalizeLocale(locale)
	if err != nil {
		normalized = catalog.defaultLocale
	}
	if values := catalog.messages[normalized]; values != nil {
		if value, exists := values[key]; exists {
			return value, nil
		}
	}
	if value, exists := catalog.messages[catalog.defaultLocale][key]; exists {
		return value, nil
	}
	return "", fmt.Errorf("resolve message: key %q is not defined", key)
}

func catalogLocale(file, defaultLocale string) (string, error) {
	base := path.Base(file)
	if !strings.HasSuffix(base, ".properties") {
		return "", fmt.Errorf(
			"parse message catalog %q: filename must end in .properties",
			file,
		)
	}
	stem := strings.TrimSuffix(base, ".properties")
	separator := strings.LastIndexByte(stem, '_')
	if separator < 0 {
		return defaultLocale, nil
	}
	locale, err := normalizeLocale(strings.ReplaceAll(stem[separator+1:], "_", "-"))
	if err != nil {
		return "", fmt.Errorf("parse message catalog %q: locale: %w", file, err)
	}
	return locale, nil
}

func validateCatalogPattern(pattern string) error {
	if pattern == "" ||
		strings.TrimSpace(pattern) != pattern ||
		strings.HasPrefix(pattern, "/") ||
		strings.Contains(pattern, "\\") ||
		slices.Contains(strings.Split(pattern, "/"), "..") {
		return errors.New(
			"parse message catalog: pattern must be a relative slash path",
		)
	}
	return nil
}

func loadCatalogFile(
	source fs.FS,
	file string,
	defaultLocale string,
	remaining *int64,
) (string, map[string]string, error) {
	locale, err := catalogLocale(file, defaultLocale)
	if err != nil {
		return "", nil, err
	}
	content, err := readCatalogFile(source, file, remaining)
	if err != nil {
		return "", nil, err
	}
	values, err := parseProperties(file, string(content))
	if err != nil {
		return "", nil, err
	}
	return locale, values, nil
}

func normalizeLocale(value string) (string, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), "_", "-")
	if value == "" {
		return "", errors.New("locale is empty")
	}
	parts := strings.Split(value, "-")
	for index, part := range parts {
		if len(part) < 1 || len(part) > 8 {
			return "", fmt.Errorf("locale %q is invalid", value)
		}
		for _, character := range part {
			if (character < 'a' || character > 'z') &&
				(character < 'A' || character > 'Z') &&
				(character < '0' || character > '9') {
				return "", fmt.Errorf("locale %q is invalid", value)
			}
		}
		parts[index] = strings.ToLower(part)
	}
	return strings.Join(parts, "-"), nil
}
