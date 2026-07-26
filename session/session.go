// Package session provides typed, encrypted, stateless HTTP sessions with
// explicit key rotation and secure cookie defaults.
package session

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const (
	defaultCookieName       = "spice_session"
	defaultLifetime         = 8 * time.Hour
	defaultClockSkew        = time.Minute
	maxLifetime             = 30 * 24 * time.Hour
	maxClockSkew            = 5 * time.Minute
	maxSessionPayloadBytes  = 2048
	maxCookieBytes          = 4096
	maxKeyIDBytes           = 64
	maxDecryptionKeys       = 8
	aes256KeyBytes          = 32
	sessionTokenVersion     = "v1"
	sessionTokenPartCount   = 3
	cookieDeletionUnixEpoch = 1
)

var (
	// ErrNotFound means the request carries no configured session cookie.
	ErrNotFound = errors.New("session not found")
	// ErrInvalid means a cookie is malformed, unauthenticated, or violates the
	// configured bounded session contract.
	ErrInvalid = errors.New("session is invalid")
	// ErrExpired means an authenticated session reached its embedded expiry.
	ErrExpired = errors.New("session expired")
)

// Key is one AES-256-GCM key. The first configured key seals new sessions;
// remaining keys only decrypt sessions during bounded rotation.
type Key struct {
	ID     string
	Secret []byte
}

// Options configures one immutable session manager.
type Options struct {
	Name          string
	Path          string
	Domain        string
	Lifetime      time.Duration
	ClockSkew     time.Duration
	SameSite      http.SameSite
	AllowInsecure bool
	Keys          []Key
}

// Record is one authenticated session value and its server-issued metadata.
type Record[T any] struct {
	Value        T
	IssuedAt     time.Time
	ExpiresAt    time.Time
	NeedsRefresh bool
}

// Manager seals and loads one exact session value type. It is safe for
// concurrent use.
type Manager[T any] struct {
	cookie    cookieSettings
	lifetime  time.Duration
	clockSkew time.Duration
	primary   string
	keys      map[string]cipher.AEAD
}

type cookieSettings struct {
	name     string
	path     string
	domain   string
	secure   bool
	sameSite http.SameSite
}

type envelope struct {
	IssuedAt  int64           `json:"iat"`
	ExpiresAt int64           `json:"exp"`
	Data      json.RawMessage `json:"data"`
}

type sealedSession struct {
	value     string
	expiresAt time.Time
}

// New validates and freezes a typed session manager without reading process
// state or generating a token.
func New[T any](options Options) (*Manager[T], error) {
	normalized, err := normalizeOptions(options)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]cipher.AEAD, len(normalized.Keys))
	for index, key := range normalized.Keys {
		if !validKeyID(key.ID) {
			return nil, fmt.Errorf("construct session manager: key %d ID is invalid", index)
		}
		if len(key.Secret) != aes256KeyBytes {
			return nil, fmt.Errorf(
				"construct session manager: key %q must contain exactly %d bytes",
				key.ID,
				aes256KeyBytes,
			)
		}
		if _, duplicate := keys[key.ID]; duplicate {
			return nil, fmt.Errorf(
				"construct session manager: key ID %q is duplicated",
				key.ID,
			)
		}
		block, blockErr := aes.NewCipher(append([]byte(nil), key.Secret...))
		if blockErr != nil {
			return nil, fmt.Errorf("construct session manager: key %q: %w", key.ID, blockErr)
		}
		authenticated, authenticatedErr := cipher.NewGCM(block)
		if authenticatedErr != nil {
			return nil, fmt.Errorf(
				"construct session manager: key %q: %w",
				key.ID,
				authenticatedErr,
			)
		}
		keys[key.ID] = authenticated
	}
	return &Manager[T]{
		cookie:    newCookieSettings(normalized),
		lifetime:  normalized.Lifetime,
		clockSkew: normalized.ClockSkew,
		primary:   normalized.Keys[0].ID,
		keys:      keys,
	}, nil
}

// Save seals value and appends one Set-Cookie header. It never logs or returns
// plaintext session data.
func (manager *Manager[T]) Save(
	writer http.ResponseWriter,
	value T,
	now time.Time,
) error {
	if nilInterface(writer) {
		return errors.New("save session: response writer is nil")
	}
	sealed, err := manager.seal(value, now)
	if err != nil {
		return err
	}
	return manager.cookie.use(
		sealed.value,
		sealed.expiresAt,
		int(manager.lifetime/time.Second),
		func(cookie *http.Cookie) error {
			if len(cookie.String()) > maxCookieBytes {
				return errors.New("save session: cookie exceeds 4096 bytes")
			}
			http.SetCookie(writer, cookie)
			return nil
		},
	)
}

// Load authenticates and strictly decodes the configured request cookie.
func (manager *Manager[T]) Load(
	request *http.Request,
	now time.Time,
) (Record[T], error) {
	var zero Record[T]
	if manager == nil || len(manager.keys) == 0 {
		return zero, errors.New("load session: manager is nil")
	}
	if request == nil {
		return zero, errors.New("load session: request is nil")
	}
	if now.IsZero() {
		return zero, errors.New("load session: current time is required")
	}
	cookie, err := uniqueCookie(request, manager.cookie.name)
	if err != nil {
		return zero, err
	}
	stored, keyID, err := manager.open(cookie.Value)
	if err != nil {
		return zero, err
	}
	issuedAt, expiresAt, err := sessionTimes(stored, now, manager.clockSkew)
	if err != nil {
		return zero, err
	}
	if len(stored.Data) == 0 ||
		len(stored.Data) > maxSessionPayloadBytes ||
		bytes.Equal(bytes.TrimSpace(stored.Data), []byte("null")) {
		return zero, invalidSession()
	}
	var value T
	if err := strictJSON(stored.Data, &value); err != nil {
		return zero, invalidSession()
	}
	return Record[T]{
		Value:        value,
		IssuedAt:     issuedAt,
		ExpiresAt:    expiresAt,
		NeedsRefresh: keyID != manager.primary,
	}, nil
}

// Clear appends a cookie deletion header with the manager's exact security and
// scope attributes.
func (manager *Manager[T]) Clear(writer http.ResponseWriter) error {
	if manager == nil || len(manager.keys) == 0 {
		return errors.New("clear session: manager is nil")
	}
	if nilInterface(writer) {
		return errors.New("clear session: response writer is nil")
	}
	return manager.cookie.use(
		"",
		time.Unix(cookieDeletionUnixEpoch, 0).UTC(),
		-1,
		func(cookie *http.Cookie) error {
			http.SetCookie(writer, cookie)
			return nil
		},
	)
}

func (manager *Manager[T]) seal(value T, now time.Time) (sealedSession, error) {
	if manager == nil || len(manager.keys) == 0 {
		return sealedSession{}, errors.New("save session: manager is nil")
	}
	if now.IsZero() {
		return sealedSession{}, errors.New("save session: current time is required")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return sealedSession{}, fmt.Errorf("save session: encode value: %w", err)
	}
	if len(data) == 0 ||
		len(data) > maxSessionPayloadBytes ||
		bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return sealedSession{}, fmt.Errorf(
			"save session: encoded value must contain between 1 and %d non-null bytes",
			maxSessionPayloadBytes,
		)
	}
	return manager.sealData(data, now)
}

func (manager *Manager[T]) sealData(data []byte, now time.Time) (sealedSession, error) {
	issuedAt := now.UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(manager.lifetime)
	plaintext, err := json.Marshal(envelope{
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: expiresAt.Unix(),
		Data:      append(json.RawMessage(nil), data...),
	})
	if err != nil {
		return sealedSession{}, fmt.Errorf("save session: encode envelope: %w", err)
	}
	authenticated, exists := manager.keys[manager.primary]
	if !exists || authenticated == nil {
		return sealedSession{}, errors.New("save session: manager has no primary key")
	}
	nonce := make([]byte, authenticated.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return sealedSession{}, fmt.Errorf("save session: generate nonce: %w", err)
	}
	sealed := authenticated.Seal(
		append([]byte(nil), nonce...),
		nonce,
		plaintext,
		manager.associatedData(manager.primary),
	)
	value := strings.Join([]string{
		sessionTokenVersion,
		manager.primary,
		base64.RawURLEncoding.EncodeToString(sealed),
	}, ".")
	return sealedSession{value: value, expiresAt: expiresAt}, nil
}

func (manager *Manager[T]) associatedData(keyID string) []byte {
	return []byte(sessionTokenVersion + "\x00" + manager.cookie.name + "\x00" + keyID)
}

func (manager *Manager[T]) open(value string) (envelope, string, error) {
	parts := strings.Split(value, ".")
	if len(parts) != sessionTokenPartCount ||
		parts[0] != sessionTokenVersion ||
		!validKeyID(parts[1]) {
		return envelope{}, "", invalidSession()
	}
	authenticated, exists := manager.keys[parts[1]]
	if !exists {
		return envelope{}, "", invalidSession()
	}
	sealed, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil ||
		len(sealed) <= authenticated.NonceSize()+authenticated.Overhead() ||
		len(sealed) > maxCookieBytes {
		return envelope{}, "", invalidSession()
	}
	nonce := sealed[:authenticated.NonceSize()]
	plaintext, err := authenticated.Open(
		nil,
		nonce,
		sealed[authenticated.NonceSize():],
		manager.associatedData(parts[1]),
	)
	if err != nil {
		return envelope{}, "", invalidSession()
	}
	var stored envelope
	if err := strictJSON(plaintext, &stored); err != nil {
		return envelope{}, "", invalidSession()
	}
	return stored, parts[1], nil
}

func sessionTimes(
	stored envelope,
	now time.Time,
	clockSkew time.Duration,
) (time.Time, time.Time, error) {
	issuedAt := time.Unix(stored.IssuedAt, 0).UTC()
	expiresAt := time.Unix(stored.ExpiresAt, 0).UTC()
	if stored.IssuedAt <= 0 ||
		stored.ExpiresAt <= stored.IssuedAt ||
		expiresAt.Sub(issuedAt) > maxLifetime ||
		now.UTC().Add(clockSkew).Before(issuedAt) {
		return time.Time{}, time.Time{}, invalidSession()
	}
	if !now.UTC().Before(expiresAt) {
		return time.Time{}, time.Time{}, fmt.Errorf("load session: %w", ErrExpired)
	}
	return issuedAt, expiresAt, nil
}

func normalizeOptions(options Options) (Options, error) {
	options = sessionDefaults(options)
	if err := validateSessionDurations(options); err != nil {
		return Options{}, err
	}
	if err := validateSameSite(options); err != nil {
		return Options{}, err
	}
	if err := validateCookieOptions(options); err != nil {
		return Options{}, err
	}
	return options, nil
}

func sessionDefaults(options Options) Options {
	if options.Name == "" {
		options.Name = defaultCookieName
	}
	if options.Path == "" {
		options.Path = "/"
	}
	if options.Lifetime == 0 {
		options.Lifetime = defaultLifetime
	}
	if options.ClockSkew == 0 {
		options.ClockSkew = defaultClockSkew
	}
	if options.SameSite == 0 || options.SameSite == http.SameSiteDefaultMode {
		options.SameSite = http.SameSiteLaxMode
	}
	return options
}

func validateSessionDurations(options Options) error {
	if len(options.Keys) < 1 || len(options.Keys) > maxDecryptionKeys {
		return fmt.Errorf(
			"construct session manager: keys must contain between 1 and %d entries",
			maxDecryptionKeys,
		)
	}
	if options.Lifetime < time.Minute ||
		options.Lifetime > maxLifetime ||
		options.Lifetime%time.Second != 0 {
		return errors.New(
			"construct session manager: lifetime must be a whole number of seconds between 1m and 720h",
		)
	}
	if options.ClockSkew < 0 ||
		options.ClockSkew > maxClockSkew ||
		options.ClockSkew%time.Second != 0 {
		return errors.New(
			"construct session manager: clock skew must be a whole number of seconds between 0 and 5m",
		)
	}
	return nil
}

func validateSameSite(options Options) error {
	switch options.SameSite {
	case http.SameSiteLaxMode, http.SameSiteStrictMode:
	case http.SameSiteNoneMode:
		if options.AllowInsecure {
			return errors.New(
				"construct session manager: SameSite=None requires secure cookies",
			)
		}
	case http.SameSiteDefaultMode:
		return errors.New("construct session manager: SameSite mode is invalid")
	default:
		return errors.New("construct session manager: SameSite mode is invalid")
	}
	return nil
}

func validateCookieOptions(options Options) error {
	err := newCookieSettings(options).use(
		"validation",
		time.Time{},
		0,
		func(cookie *http.Cookie) error {
			return cookie.Valid()
		},
	)
	if err != nil {
		return fmt.Errorf("construct session manager: cookie options: %w", err)
	}
	if strings.HasPrefix(options.Name, "__Host-") &&
		(options.AllowInsecure || options.Path != "/" || options.Domain != "") {
		return errors.New(
			"construct session manager: __Host- cookies require Secure, Path=/, and no Domain",
		)
	}
	if strings.HasPrefix(options.Name, "__Secure-") && options.AllowInsecure {
		return errors.New(
			"construct session manager: __Secure- cookies require Secure",
		)
	}
	return nil
}

func newCookieSettings(options Options) cookieSettings {
	return cookieSettings{
		name:     options.Name,
		path:     options.Path,
		domain:   options.Domain,
		secure:   !options.AllowInsecure,
		sameSite: options.SameSite,
	}
}

func (settings cookieSettings) use(
	value string,
	expires time.Time,
	maxAge int,
	consumer func(*http.Cookie) error,
) error {
	// #nosec G124 -- Secure may be false only after validated AllowInsecure;
	// HttpOnly and an explicit validated SameSite mode remain mandatory.
	cookie := &http.Cookie{
		Name:     settings.name,
		Value:    value,
		Path:     settings.path,
		Domain:   settings.domain,
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   settings.secure,
		SameSite: settings.sameSite,
	}
	return consumer(cookie)
}

func uniqueCookie(request *http.Request, name string) (*http.Cookie, error) {
	var found *http.Cookie
	for _, cookie := range request.Cookies() {
		if cookie.Name != name {
			continue
		}
		if found != nil {
			return nil, invalidSession()
		}
		found = cookie
	}
	if found == nil {
		return nil, fmt.Errorf("load session: %w", ErrNotFound)
	}
	if found.Value == "" || len(found.Value) > maxCookieBytes {
		return nil, invalidSession()
	}
	return found, nil
}

func strictJSON(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func validKeyID(value string) bool {
	if len(value) < 1 || len(value) > maxKeyIDBytes {
		return false
	}
	for _, character := range []byte(value) {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '_' ||
			character == '-' {
			continue
		}
		return false
	}
	return true
}

func invalidSession() error {
	return fmt.Errorf("load session: %w", ErrInvalid)
}

func nilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	kind := reflected.Kind()
	nilCapable := kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.Interface ||
		kind == reflect.Map ||
		kind == reflect.Pointer ||
		kind == reflect.Slice
	return nilCapable && reflected.IsNil()
}
