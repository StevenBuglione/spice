package outbox

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spice-framework/spice/data"
)

var errTest = errors.New("test failure")

func TestMessageAndDeliveryAreValidatedAndImmutable(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"order":"1"}`)
	occurredAt := time.Date(2026, time.July, 25, 10, 0, 0, 0, time.FixedZone("test", 3600))
	message, err := NewMessage(MessageSpec{
		ID:          "message-1",
		Topic:       "orders.OrderPlaced",
		Module:      "example.com/shop/orders",
		ContentType: "application/json; charset=utf-8",
		Payload:     payload,
		OccurredAt:  occurredAt,
	})
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}
	payload[0] = '!'
	returned := message.Payload()
	returned[0] = '!'
	if message.ID() != "message-1" ||
		message.Topic() != "orders.OrderPlaced" ||
		message.Module() != "example.com/shop/orders" ||
		message.ContentType() != "application/json; charset=utf-8" ||
		string(message.Payload()) != `{"order":"1"}` ||
		message.OccurredAt().Location() != time.UTC {
		t.Fatalf("message = %#v", message)
	}
	delivery, err := NewDelivery(message, "lease-1", 2)
	if err != nil {
		t.Fatalf("NewDelivery() error = %v", err)
	}
	if delivery.Message().ID() != message.ID() ||
		delivery.Receipt() != "lease-1" ||
		delivery.Attempt() != 2 {
		t.Fatalf("delivery = %#v", delivery)
	}
}

func TestDispatcherPublishesAndCompletesInStoreOrder(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	first := testDelivery(t, "message-1", "lease-1", now, 1)
	second := testDelivery(t, "message-2", "lease-2", now.Add(time.Second), 1)
	store := &fakeStore{deliveries: []Delivery{first, second}}
	var published []string
	publisher := publisherFunc(func(_ context.Context, message Message) error {
		published = append(published, message.ID())
		return nil
	})
	var observations []Observation
	dispatcher := newTestDispatcher(t, store, publisher, Options{
		Owner:     "worker-1",
		BatchSize: 10,
		Lease:     time.Minute,
		Clock:     func() time.Time { return now },
	}, func(_ context.Context, observation Observation) {
		observations = append(observations, observation)
	})
	result, err := dispatcher.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result != (Result{Claimed: 2, Published: 2, Completed: 2}) ||
		!slices.Equal(published, []string{"message-1", "message-2"}) ||
		!slices.Equal(store.completed, []string{"lease-1", "lease-2"}) {
		t.Fatalf("result = %#v, published = %v, completed = %v", result, published, store.completed)
	}
	if len(store.claims) != 1 || store.claims[0] != (ClaimRequest{
		Owner: "worker-1",
		Now:   now,
		Lease: time.Minute,
		Limit: 10,
	}) {
		t.Fatalf("claims = %#v", store.claims)
	}
	if len(observations) != 2 ||
		!observations[0].Published ||
		!observations[0].Completed ||
		observations[0].Released ||
		observations[0].Err != nil ||
		observations[0].Duration < 0 {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestDispatcherReleasesFailuresAndContinues(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{deliveries: []Delivery{
		testDelivery(t, "message-1", "lease-1", now, 3),
		testDelivery(t, "message-2", "lease-2", now.Add(time.Second), 1),
	}}
	publisher := publisherFunc(func(_ context.Context, message Message) error {
		if message.ID() == "message-1" {
			return errTest
		}
		return nil
	})
	var observations []Observation
	dispatcher := newTestDispatcher(t, store, publisher, Options{
		Owner:     "worker-1",
		BatchSize: 2,
		Lease:     time.Minute,
		Clock:     func() time.Time { return now },
		FailureDelay: func(delivery Delivery) time.Duration {
			return time.Duration(delivery.Attempt()) * time.Minute
		},
	}, func(_ context.Context, observation Observation) {
		observations = append(observations, observation)
	})
	result, err := dispatcher.RunOnce(context.Background())
	if !errors.Is(err, errTest) ||
		result != (Result{Claimed: 2, Published: 1, Completed: 1, Released: 1}) {
		t.Fatalf("RunOnce() = %#v, %v", result, err)
	}
	if len(store.released) != 1 ||
		store.released[0].Receipt != "lease-1" ||
		!store.released[0].AvailableAt.Equal(now.Add(3*time.Minute)) ||
		!slices.Equal(store.completed, []string{"lease-2"}) {
		t.Fatalf("released = %#v, completed = %v", store.released, store.completed)
	}
	if len(observations) != 2 ||
		observations[0].Published ||
		!observations[0].Released ||
		!errors.Is(observations[0].Err, errTest) ||
		!observations[1].Completed {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestDispatcherReportsLeaseTransitionFailures(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	tests := []struct {
		name      string
		publish   error
		store     *fakeStore
		published int
	}{
		{
			name:    "release",
			publish: errTest,
			store: &fakeStore{
				releaseErr: errors.New("release failed"),
			},
		},
		{
			name: "complete",
			store: &fakeStore{
				completeErr: errors.New("complete failed"),
			},
			published: 1,
		},
	}
	for _, test := range tests {
		test.store.deliveries = []Delivery{testDelivery(t, "message-1", "lease-1", now, 1)}
		dispatcher := newTestDispatcher(t, test.store, publisherFunc(
			func(context.Context, Message) error { return test.publish },
		), testOptions(now))
		result, err := dispatcher.RunOnce(context.Background())
		if err == nil || result.Published != test.published || result.Completed != 0 {
			t.Fatalf("%s: RunOnce() = %#v, %v", test.name, result, err)
		}
		if test.name == "release" && !errors.Is(err, errTest) {
			t.Fatalf("%s: RunOnce() error = %v", test.name, err)
		}
	}
}

func TestDispatcherObservesAndRepanicsPublisherPanic(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := &fakeStore{deliveries: []Delivery{
		testDelivery(t, "message-1", "lease-1", now, 1),
	}}
	var observation Observation
	dispatcher := newTestDispatcher(t, store, publisherFunc(
		func(context.Context, Message) error {
			panic("publisher panic")
		},
	), testOptions(now), func(_ context.Context, result Observation) {
		observation = result
	})
	defer func() {
		if recovered := recover(); recovered != "publisher panic" {
			t.Fatalf("panic = %#v", recovered)
		}
		if !observation.Panicked || !errors.Is(observation.Err, ErrPublisherPanicked) {
			t.Fatalf("observation = %#v", observation)
		}
	}()
	if _, err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	t.Fatal("RunOnce() did not panic")
}

func TestDispatcherRejectsInvalidStoreResults(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	first := testDelivery(t, "message-1", "lease-1", now, 1)
	second := testDelivery(t, "message-2", "lease-2", now.Add(time.Second), 1)
	duplicateID, err := NewDelivery(first.Message(), "lease-2", 1)
	if err != nil {
		t.Fatalf("NewDelivery() error = %v", err)
	}
	duplicateReceipt, err := NewDelivery(second.Message(), "lease-1", 1)
	if err != nil {
		t.Fatalf("NewDelivery() error = %v", err)
	}
	tests := [][]Delivery{
		{first, second},
		{Delivery{}},
		{first, duplicateID},
		{first, duplicateReceipt},
		{second, first},
	}
	for index, deliveries := range tests {
		limit := 2
		if index == 0 {
			limit = 1
		}
		store := &fakeStore{deliveries: deliveries}
		dispatcher := newTestDispatcher(t, store, publisherFunc(
			func(context.Context, Message) error {
				t.Fatal("publisher called for invalid store result")
				return nil
			},
		), Options{
			Owner:     "worker-1",
			BatchSize: limit,
			Lease:     time.Minute,
			Clock:     func() time.Time { return now },
		})
		if _, err := dispatcher.RunOnce(context.Background()); err == nil {
			t.Fatalf("RunOnce(case %d) error = nil", index)
		}
	}
}

func TestConstructionAndExecutionValidation(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	invalidMessages := []MessageSpec{
		{},
		{ID: " id", Topic: "topic", Module: "module", ContentType: "application/json", Payload: []byte("{}"), OccurredAt: now},
		{ID: "id", Topic: "", Module: "module", ContentType: "application/json", Payload: []byte("{}"), OccurredAt: now},
		{ID: "id", Topic: "topic", Module: "", ContentType: "application/json", Payload: []byte("{}"), OccurredAt: now},
		{ID: "id", Topic: "topic", Module: "module", ContentType: "bad", Payload: []byte("{}"), OccurredAt: now},
		{ID: "id", Topic: "topic", Module: "module", ContentType: "application/json", OccurredAt: now},
		{ID: "id", Topic: "topic", Module: "module", ContentType: "application/json", Payload: []byte("{}")},
	}
	for index, spec := range invalidMessages {
		if _, err := NewMessage(spec); err == nil {
			t.Fatalf("NewMessage(case %d) error = nil", index)
		}
	}
	message := testMessage(t, "message-1", now)
	if _, err := NewDelivery(message, "", 1); err == nil {
		t.Fatal("NewDelivery(no receipt) error = nil")
	}
	if _, err := NewDelivery(message, "lease", 0); err == nil {
		t.Fatal("NewDelivery(no attempt) error = nil")
	}
	store := &fakeStore{}
	publisher := publisherFunc(func(context.Context, Message) error { return nil })
	invalidOptions := []Options{
		{},
		{Owner: " worker", BatchSize: 1, Lease: time.Second},
		{Owner: "worker", BatchSize: 0, Lease: time.Second},
		{Owner: "worker", BatchSize: 1001, Lease: time.Second},
		{Owner: "worker", BatchSize: 1},
		{Owner: "worker", BatchSize: 1, Lease: maxFailureDelay + 1},
	}
	for index, options := range invalidOptions {
		if _, err := NewDispatcher(store, publisher, options); err == nil {
			t.Fatalf("NewDispatcher(case %d) error = nil", index)
		}
	}
	if _, err := NewDispatcher(nil, publisher, testOptions(now)); err == nil {
		t.Fatal("NewDispatcher(nil store) error = nil")
	}
	if _, err := NewDispatcher(store, nil, testOptions(now)); err == nil {
		t.Fatal("NewDispatcher(nil publisher) error = nil")
	}
	var typedNilStore *fakeStore
	if _, err := NewDispatcher(typedNilStore, publisher, testOptions(now)); err == nil {
		t.Fatal("NewDispatcher(typed nil store) error = nil")
	}
	var typedNilPublisher publisherFunc
	if _, err := NewDispatcher(store, typedNilPublisher, testOptions(now)); err == nil {
		t.Fatal("NewDispatcher(typed nil publisher) error = nil")
	}
	if _, err := NewDispatcher(store, publisher, testOptions(now), nil); err == nil {
		t.Fatal("NewDispatcher(nil observer) error = nil")
	}
	if _, err := (*Dispatcher)(nil).RunOnce(context.Background()); err == nil {
		t.Fatal("nil RunOnce() error = nil")
	}
	dispatcher := newTestDispatcher(t, store, publisher, testOptions(now))
	if _, err := dispatcher.RunOnce(nilContext()); err == nil {
		t.Fatal("RunOnce(nil context) error = nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := dispatcher.RunOnce(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("RunOnce(canceled) error = %v", err)
	}
}

type publisherFunc func(context.Context, Message) error

func (publisher publisherFunc) Publish(ctx context.Context, message Message) error {
	return publisher(ctx, message)
}

type fakeStore struct {
	deliveries  []Delivery
	claimErr    error
	completeErr error
	releaseErr  error
	claims      []ClaimRequest
	completed   []string
	released    []Release
}

func (*fakeStore) Enqueue(context.Context, data.Executor, Message) error {
	return nil
}

func (store *fakeStore) Claim(_ context.Context, request ClaimRequest) ([]Delivery, error) {
	store.claims = append(store.claims, request)
	return append([]Delivery(nil), store.deliveries...), store.claimErr
}

func (store *fakeStore) Complete(_ context.Context, completion Completion) error {
	if store.completeErr == nil {
		store.completed = append(store.completed, completion.Receipt)
	}
	return store.completeErr
}

func (store *fakeStore) Release(_ context.Context, release Release) error {
	if store.releaseErr == nil {
		store.released = append(store.released, release)
	}
	return store.releaseErr
}

func newTestDispatcher(
	t *testing.T,
	store Store,
	publisher Publisher,
	options Options,
	observers ...Observer,
) *Dispatcher {
	t.Helper()
	dispatcher, err := NewDispatcher(store, publisher, options, observers...)
	if err != nil {
		t.Fatalf("NewDispatcher() error = %v", err)
	}
	return dispatcher
}

func testOptions(now time.Time) Options {
	return Options{
		Owner:     "worker-1",
		BatchSize: 10,
		Lease:     time.Minute,
		Clock:     func() time.Time { return now },
	}
}

func testDelivery(
	t *testing.T,
	id string,
	receipt string,
	occurredAt time.Time,
	attempt int,
) Delivery {
	t.Helper()
	delivery, err := NewDelivery(testMessage(t, id, occurredAt), receipt, attempt)
	if err != nil {
		t.Fatalf("NewDelivery() error = %v", err)
	}
	return delivery
}

func testMessage(t *testing.T, id string, occurredAt time.Time) Message {
	t.Helper()
	message, err := NewMessage(MessageSpec{
		ID:          id,
		Topic:       "orders.OrderPlaced",
		Module:      "example.com/shop/orders",
		ContentType: "application/json",
		Payload:     []byte(`{"order":"1"}`),
		OccurredAt:  occurredAt,
	})
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}
	return message
}

func nilContext() context.Context {
	return nil
}

func TestErrorMessagesExcludePayloadAndReceipt(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := &fakeStore{
		deliveries: []Delivery{
			testDelivery(t, "message-1", "secret-receipt", now, 1),
		},
		releaseErr: errTest,
	}
	dispatcher := newTestDispatcher(t, store, publisherFunc(
		func(context.Context, Message) error { return errTest },
	), testOptions(now))
	_, err := dispatcher.RunOnce(context.Background())
	if err == nil ||
		strings.Contains(err.Error(), `{"order":"1"}`) ||
		strings.Contains(err.Error(), "secret-receipt") {
		t.Fatalf("RunOnce() error = %v", err)
	}
}
