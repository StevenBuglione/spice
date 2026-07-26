package session

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

var sessionNow = time.Date(2026, time.July, 26, 15, 0, 0, 0, time.UTC)

type testSessionValue struct {
	Subject string   `json:"subject"`
	Roles   []string `json:"roles"`
	Data    string   `json:"data,omitempty"`
}

func TestManagerRoundTripUsesSecureBoundedCookies(t *testing.T) {
	t.Parallel()

	manager := newTestManager[testSessionValue](t, Options{
		Name: "__Host-spice",
		Keys: []Key{testSessionKey("current", 'a')},
	})
	value := testSessionValue{
		Subject: "account-41",
		Roles:   []string{"orders.read", "orders.write"},
	}
	first := saveTestSession(t, manager, value, sessionNow)
	second := saveTestSession(t, manager, value, sessionNow)
	if first.Value == second.Value {
		t.Fatal("independent session seals reused ciphertext")
	}
	if strings.Contains(first.Value, value.Subject) ||
		strings.Contains(first.Value, value.Roles[0]) {
		t.Fatal("session cookie exposed plaintext")
	}
	if !first.Secure ||
		!first.HttpOnly ||
		first.Path != "/" ||
		first.Domain != "" ||
		first.SameSite != http.SameSiteLaxMode ||
		first.MaxAge != int(defaultLifetime/time.Second) ||
		!first.Expires.Equal(sessionNow.Add(defaultLifetime)) {
		t.Fatalf("cookie attributes = %#v", first)
	}

	request := httptest.NewRequest(http.MethodGet, "https://example.test/orders", nil)
	request.AddCookie(first)
	record, err := manager.Load(request, sessionNow.Add(time.Minute))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if record.Value.Subject != value.Subject ||
		!slices.Equal(record.Value.Roles, value.Roles) ||
		!record.IssuedAt.Equal(sessionNow) ||
		!record.ExpiresAt.Equal(sessionNow.Add(defaultLifetime)) ||
		record.NeedsRefresh {
		t.Fatalf("record = %#v", record)
	}
}

func TestManagerSupportsKeyRotationAndRejectsTampering(t *testing.T) {
	t.Parallel()

	oldManager := newTestManager[testSessionValue](t, Options{
		Keys: []Key{testSessionKey("old", 'o')},
	})
	oldCookie := saveTestSession(
		t,
		oldManager,
		testSessionValue{Subject: "account-41"},
		sessionNow,
	)
	rotated := newTestManager[testSessionValue](t, Options{
		Keys: []Key{
			testSessionKey("current", 'n'),
			testSessionKey("old", 'o'),
		},
	})
	record, err := rotated.Load(requestWithCookie(oldCookie), sessionNow)
	if err != nil {
		t.Fatalf("load rotated session: %v", err)
	}
	if !record.NeedsRefresh || record.Value.Subject != "account-41" {
		t.Fatalf("rotated record = %#v", record)
	}

	tampered := *oldCookie
	tampered.Value = mutateSessionToken(tampered.Value)
	if _, err := rotated.Load(
		requestWithCookie(&tampered),
		sessionNow,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("tampered Load() error = %v", err)
	}

	wrongName := newTestManager[testSessionValue](t, Options{
		Name: "other_session",
		Keys: []Key{testSessionKey("old", 'o')},
	})
	renamed := *oldCookie
	renamed.Name = "other_session"
	if _, err := wrongName.Load(
		requestWithCookie(&renamed),
		sessionNow,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("wrong-name Load() error = %v", err)
	}

	unknownKey := newTestManager[testSessionValue](t, Options{
		Keys: []Key{testSessionKey("different", 'd')},
	})
	if _, err := unknownKey.Load(
		requestWithCookie(oldCookie),
		sessionNow,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown-key Load() error = %v", err)
	}
}

func TestManagerRejectsMissingDuplicateExpiredAndFutureSessions(t *testing.T) {
	t.Parallel()

	manager := newTestManager[testSessionValue](t, Options{
		Lifetime: time.Hour,
		Keys:     []Key{testSessionKey("current", 'a')},
	})
	if _, err := manager.Load(
		httptest.NewRequest(http.MethodGet, "https://example.test", nil),
		sessionNow,
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing Load() error = %v", err)
	}
	cookie := saveTestSession(
		t,
		manager,
		testSessionValue{Subject: "account-41"},
		sessionNow,
	)
	duplicate := requestWithCookie(cookie)
	duplicate.AddCookie(cookie)
	if _, err := manager.Load(
		duplicate,
		sessionNow,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate Load() error = %v", err)
	}
	if _, err := manager.Load(
		requestWithCookie(cookie),
		sessionNow.Add(time.Hour),
	); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired Load() error = %v", err)
	}

	future := saveTestSession(
		t,
		manager,
		testSessionValue{Subject: "account-42"},
		sessionNow.Add(10*time.Minute),
	)
	if _, err := manager.Load(
		requestWithCookie(future),
		sessionNow,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("future Load() error = %v", err)
	}
}

func TestManagerStrictlyBoundsEncodingAndDecoding(t *testing.T) {
	t.Parallel()

	manager := newTestManager[testSessionValue](t, Options{
		Keys: []Key{testSessionKey("current", 'a')},
	})
	if err := manager.Save(
		httptest.NewRecorder(),
		testSessionValue{Data: strings.Repeat("x", maxSessionPayloadBytes)},
		sessionNow,
	); err == nil {
		t.Fatal("Save(oversized) unexpectedly succeeded")
	}
	nullManager := newTestManager[*testSessionValue](t, Options{
		Keys: []Key{testSessionKey("current", 'a')},
	})
	if err := nullManager.Save(
		httptest.NewRecorder(),
		nil,
		sessionNow,
	); err == nil {
		t.Fatal("Save(nil) unexpectedly succeeded")
	}
	unencodable := newTestManager[struct {
		Channel chan int `json:"channel"`
	}](t, Options{
		Keys: []Key{testSessionKey("current", 'a')},
	})
	if err := unencodable.Save(
		httptest.NewRecorder(),
		struct {
			Channel chan int `json:"channel"`
		}{Channel: make(chan int)},
		sessionNow,
	); err == nil {
		t.Fatal("Save(unencodable) unexpectedly succeeded")
	}

	unknownField, err := manager.sealData(
		[]byte(`{"subject":"account-41","roles":[],"unexpected":true}`),
		sessionNow,
	)
	if err != nil {
		t.Fatalf("seal unknown-field value: %v", err)
	}
	if _, err := manager.Load(
		requestWithCookie(&http.Cookie{
			Name:  defaultCookieName,
			Value: unknownField.value,
		}),
		sessionNow,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown-field Load() error = %v", err)
	}

	for _, value := range []string{
		"",
		"v1",
		"v2.current.invalid",
		"v1.invalid.id.value",
		"v1.current.%%%",
		"v1.current.YQ",
		strings.Repeat("x", maxCookieBytes+1),
	} {
		cookie := &http.Cookie{Name: defaultCookieName, Value: value}
		if _, err := manager.Load(
			requestWithCookie(cookie),
			sessionNow,
		); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Load(%q) error = %v", value, err)
		}
	}
}

func TestManagerValidatesOptions(t *testing.T) {
	t.Parallel()

	valid := Options{Keys: []Key{testSessionKey("current", 'a')}}
	tests := []struct {
		name   string
		mutate func(*Options)
	}{
		{name: "no keys", mutate: func(options *Options) { options.Keys = nil }},
		{name: "too many keys", mutate: func(options *Options) {
			options.Keys = make([]Key, maxDecryptionKeys+1)
		}},
		{name: "invalid key ID", mutate: func(options *Options) {
			options.Keys[0].ID = "bad.id"
		}},
		{name: "empty key ID", mutate: func(options *Options) {
			options.Keys[0].ID = ""
		}},
		{name: "long key ID", mutate: func(options *Options) {
			options.Keys[0].ID = strings.Repeat("x", maxKeyIDBytes+1)
		}},
		{name: "short secret", mutate: func(options *Options) {
			options.Keys[0].Secret = []byte("short")
		}},
		{name: "duplicate key ID", mutate: func(options *Options) {
			options.Keys = append(options.Keys, testSessionKey("current", 'b'))
		}},
		{name: "short lifetime", mutate: func(options *Options) {
			options.Lifetime = time.Minute - time.Second
		}},
		{name: "long lifetime", mutate: func(options *Options) {
			options.Lifetime = maxLifetime + time.Second
		}},
		{name: "fractional lifetime", mutate: func(options *Options) {
			options.Lifetime = time.Hour + time.Nanosecond
		}},
		{name: "negative skew", mutate: func(options *Options) {
			options.ClockSkew = -time.Second
		}},
		{name: "large skew", mutate: func(options *Options) {
			options.ClockSkew = maxClockSkew + time.Second
		}},
		{name: "fractional skew", mutate: func(options *Options) {
			options.ClockSkew = time.Second + time.Nanosecond
		}},
		{name: "same site", mutate: func(options *Options) {
			options.SameSite = http.SameSite(99)
		}},
		{name: "insecure same site none", mutate: func(options *Options) {
			options.SameSite = http.SameSiteNoneMode
			options.AllowInsecure = true
		}},
		{name: "invalid cookie name", mutate: func(options *Options) {
			options.Name = "bad name"
		}},
		{name: "invalid cookie path", mutate: func(options *Options) {
			options.Path = "/bad;path"
		}},
		{name: "invalid cookie domain", mutate: func(options *Options) {
			options.Domain = "bad domain"
		}},
		{name: "host prefix domain", mutate: func(options *Options) {
			options.Name = "__Host-spice"
			options.Domain = "example.test"
		}},
		{name: "host prefix path", mutate: func(options *Options) {
			options.Name = "__Host-spice"
			options.Path = "/app"
		}},
		{name: "host prefix insecure", mutate: func(options *Options) {
			options.Name = "__Host-spice"
			options.AllowInsecure = true
		}},
		{name: "secure prefix insecure", mutate: func(options *Options) {
			options.Name = "__Secure-spice"
			options.AllowInsecure = true
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			options := valid
			options.Keys = append([]Key(nil), valid.Keys...)
			options.Keys[0].Secret = append([]byte(nil), valid.Keys[0].Secret...)
			test.mutate(&options)
			if _, err := New[testSessionValue](options); err == nil {
				t.Fatal("New() unexpectedly succeeded")
			}
		})
	}
}

func TestManagerClearAndNilBoundaries(t *testing.T) {
	t.Parallel()

	manager := newTestManager[testSessionValue](t, Options{
		Name: "__Host-spice",
		Keys: []Key{testSessionKey("current", 'a')},
	})
	recorder := httptest.NewRecorder()
	if err := manager.Clear(recorder); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	response := recorder.Result()
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close clear response: %v", err)
		}
	})
	cookies := response.Cookies()
	if len(cookies) != 1 ||
		cookies[0].Name != "__Host-spice" ||
		cookies[0].Value != "" ||
		cookies[0].MaxAge != -1 ||
		!cookies[0].Secure ||
		!cookies[0].HttpOnly {
		t.Fatalf("clear cookies = %#v", cookies)
	}

	var nilWriter *httptest.ResponseRecorder
	if err := manager.Save(
		nilWriter,
		testSessionValue{},
		sessionNow,
	); err == nil {
		t.Fatal("Save(nil writer) unexpectedly succeeded")
	}
	if err := manager.Clear(nilWriter); err == nil {
		t.Fatal("Clear(nil writer) unexpectedly succeeded")
	}
	if _, err := manager.Load(nil, sessionNow); err == nil {
		t.Fatal("Load(nil request) unexpectedly succeeded")
	}
	if _, err := manager.Load(
		httptest.NewRequest(http.MethodGet, "https://example.test", nil),
		time.Time{},
	); err == nil {
		t.Fatal("Load(zero time) unexpectedly succeeded")
	}
	if err := manager.Save(
		httptest.NewRecorder(),
		testSessionValue{},
		time.Time{},
	); err == nil {
		t.Fatal("Save(zero time) unexpectedly succeeded")
	}

	var nilManager *Manager[testSessionValue]
	if err := nilManager.Save(
		httptest.NewRecorder(),
		testSessionValue{},
		sessionNow,
	); err == nil {
		t.Fatal("nil Save() unexpectedly succeeded")
	}
	if _, err := nilManager.Load(
		httptest.NewRequest(http.MethodGet, "https://example.test", nil),
		sessionNow,
	); err == nil {
		t.Fatal("nil Load() unexpectedly succeeded")
	}
	if err := nilManager.Clear(httptest.NewRecorder()); err == nil {
		t.Fatal("nil Clear() unexpectedly succeeded")
	}
}

func TestManagerIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()

	manager := newTestManager[testSessionValue](t, Options{
		Keys: []Key{testSessionKey("current", 'a')},
	})
	const workers = 32
	start := make(chan struct{})
	errorsFound := make(chan error, workers)
	var wait sync.WaitGroup
	for index := range workers {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			<-start
			value := testSessionValue{Subject: fmt.Sprintf("account-%d", worker)}
			recorder := httptest.NewRecorder()
			if err := manager.Save(recorder, value, sessionNow); err != nil {
				errorsFound <- err
				return
			}
			response := recorder.Result()
			cookies := response.Cookies()
			closeErr := response.Body.Close()
			if closeErr != nil {
				errorsFound <- closeErr
				return
			}
			if len(cookies) != 1 {
				errorsFound <- fmt.Errorf("worker %d cookies = %d", worker, len(cookies))
				return
			}
			record, err := manager.Load(requestWithCookie(cookies[0]), sessionNow)
			if err != nil {
				errorsFound <- err
				return
			}
			if record.Value.Subject != value.Subject {
				errorsFound <- fmt.Errorf(
					"worker %d subject = %q",
					worker,
					record.Value.Subject,
				)
			}
		}(index)
	}
	close(start)
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Errorf("concurrent session use: %v", err)
	}
}

func FuzzManagerLoad(f *testing.F) {
	manager, err := New[testSessionValue](Options{
		Keys: []Key{testSessionKey("current", 'a')},
	})
	if err != nil {
		f.Fatalf("New() error = %v", err)
	}
	valid := saveTestSession(
		f,
		manager,
		testSessionValue{Subject: "account-41"},
		sessionNow,
	)
	f.Add(valid.Value)
	f.Add("")
	f.Add("v1.current.invalid")
	f.Fuzz(func(t *testing.T, token string) {
		request := requestWithCookie(&http.Cookie{
			Name:  defaultCookieName,
			Value: token,
		})
		if _, loadErr := manager.Load(request, sessionNow); loadErr != nil {
			return
		}
	})
}

func ExampleManager() {
	type claims struct {
		Subject string `json:"subject"`
	}
	manager, err := New[claims](Options{
		Name: "__Host-orders",
		Keys: []Key{{
			ID:     "2026-07",
			Secret: []byte("0123456789abcdef0123456789abcdef"),
		}},
	})
	if err != nil {
		panic(err)
	}
	recorder := httptest.NewRecorder()
	if err := manager.Save(
		recorder,
		claims{Subject: "account-41"},
		sessionNow,
	); err != nil {
		panic(err)
	}
	fmt.Println(len(recorder.Header().Values("Set-Cookie")))
	// Output: 1
}

func newTestManager[T any](tb testing.TB, options Options) *Manager[T] {
	tb.Helper()
	manager, err := New[T](options)
	if err != nil {
		tb.Fatalf("New() error = %v", err)
	}
	return manager
}

func testSessionKey(id string, fill byte) Key {
	return Key{ID: id, Secret: []byte(strings.Repeat(string(fill), aes256KeyBytes))}
}

func saveTestSession[T any](
	tb testing.TB,
	manager *Manager[T],
	value T,
	now time.Time,
) *http.Cookie {
	tb.Helper()
	recorder := httptest.NewRecorder()
	if err := manager.Save(recorder, value, now); err != nil {
		tb.Fatalf("Save() error = %v", err)
	}
	response := recorder.Result()
	cookies := response.Cookies()
	if err := response.Body.Close(); err != nil {
		tb.Fatalf("close response: %v", err)
	}
	if len(cookies) != 1 {
		tb.Fatalf("saved cookies = %d", len(cookies))
	}
	return cookies[0]
}

func requestWithCookie(cookie *http.Cookie) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "https://example.test", nil)
	request.AddCookie(cookie)
	return request
}

func mutateSessionToken(value string) string {
	index := len(value) - 2
	replacement := byte('A')
	if value[index] == replacement {
		replacement = 'B'
	}
	return value[:index] + string(replacement) + value[index+1:]
}
