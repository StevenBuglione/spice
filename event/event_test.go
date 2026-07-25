package event

import (
	"context"
	"errors"
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
)

var errSubscriber = errors.New("subscriber failed")

type orderPlaced struct {
	ID string
}

func TestTopicPublishesInDeterministicOrder(t *testing.T) {
	t.Parallel()
	var calls []string
	var callsMu sync.Mutex
	handler := func(id string) Handler[orderPlaced] {
		return func(ctx context.Context, value orderPlaced) error {
			if ctx.Value(observerContextKey{}) != "observed" {
				return errors.New("observer context was not propagated")
			}
			callsMu.Lock()
			calls = append(calls, id+":"+value.ID)
			callsMu.Unlock()
			return nil
		}
	}
	observer := &recordingObserver{}
	input := []Subscriber[orderPlaced]{
		{ID: "shipping", Module: "example.com/shop/shipping", Order: 20, Handle: handler("shipping")},
		{ID: "analytics", Module: "example.com/shop/analytics", Order: 10, Handle: handler("analytics")},
		{ID: "billing", Module: "example.com/shop/billing", Order: 10, Handle: handler("billing")},
		{ID: "last", Module: "example.com/shop/last", Order: math.MaxInt, Handle: handler("last")},
		{ID: "first", Module: "example.com/shop/first", Order: math.MinInt, Handle: handler("first")},
	}
	topic, err := NewTopic(eventDefinition(), input, observer)
	if err != nil {
		t.Fatalf("NewTopic() error = %v", err)
	}
	input[0].ID = "mutated"

	if err := topic.Publish(context.Background(), orderPlaced{ID: "order-1"}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	want := []string{
		"first:order-1",
		"analytics:order-1",
		"billing:order-1",
		"shipping:order-1",
		"last:order-1",
	}
	if !slices.Equal(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	results := observer.resultsSnapshot()
	if len(results) != 5 {
		t.Fatalf("results = %#v, want 5", results)
	}
	wantIDs := []string{"first", "analytics", "billing", "shipping", "last"}
	for index, result := range results {
		if result.Interaction.Event != eventDefinition() ||
			result.Interaction.Subscriber.ID != wantIDs[index] ||
			result.Err != nil ||
			result.Panicked ||
			result.Duration < 0 {
			t.Fatalf("result %d = %#v", index, result)
		}
	}
}

func TestTopicStopsAtFirstFailure(t *testing.T) {
	t.Parallel()
	var calls []string
	topic, err := NewTopic(eventDefinition(), []Subscriber[orderPlaced]{
		{
			ID:     "first",
			Module: "example.com/shop/first",
			Handle: func(context.Context, orderPlaced) error {
				calls = append(calls, "first")
				return errSubscriber
			},
		},
		{
			ID:     "second",
			Module: "example.com/shop/second",
			Handle: func(context.Context, orderPlaced) error {
				calls = append(calls, "second")
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTopic() error = %v", err)
	}
	err = topic.Publish(context.Background(), orderPlaced{})
	if !errors.Is(err, errSubscriber) {
		t.Fatalf("Publish() error = %v, want errors.Is(errSubscriber)", err)
	}
	if want := []string{"first"}; !slices.Equal(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestTopicHonorsCancellationBetweenSubscribers(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int64
	topic, err := NewTopic(eventDefinition(), []Subscriber[orderPlaced]{
		{
			ID:     "first",
			Module: "example.com/shop/first",
			Handle: func(context.Context, orderPlaced) error {
				calls.Add(1)
				cancel()
				return nil
			},
		},
		{
			ID:     "second",
			Module: "example.com/shop/second",
			Handle: func(context.Context, orderPlaced) error {
				calls.Add(1)
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTopic() error = %v", err)
	}
	err = topic.Publish(ctx, orderPlaced{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Publish() error = %v, want context.Canceled", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("handler calls = %d, want 1", got)
	}
}

func TestTopicObservesPanicAndRepanics(t *testing.T) {
	t.Parallel()
	observer := &recordingObserver{}
	panicValue := &struct{ message string }{message: "boom"}
	topic, err := NewTopic(eventDefinition(), []Subscriber[orderPlaced]{
		{
			ID:     "panics",
			Module: "example.com/shop/listener",
			Handle: func(context.Context, orderPlaced) error {
				panic(panicValue)
			},
		},
	}, observer)
	if err != nil {
		t.Fatalf("NewTopic() error = %v", err)
	}

	recovered := recoverPublish(func() {
		publishErr := topic.Publish(context.Background(), orderPlaced{})
		t.Fatalf("Publish() error = %v, want panic", publishErr)
	})
	if recovered != panicValue {
		t.Fatalf("recovered = %#v, want original panic", recovered)
	}
	result := observer.onlyResult(t)
	if !result.Panicked || !errors.Is(result.Err, ErrPanicked) {
		t.Fatalf("result = %#v", result)
	}
}

func TestTopicValidatesConstructionAndPublish(t *testing.T) {
	t.Parallel()
	validSubscriber := Subscriber[orderPlaced]{
		ID:     "listener",
		Module: "example.com/shop/listener",
		Handle: func(context.Context, orderPlaced) error { return nil },
	}
	var typedNil *recordingObserver
	tests := []struct {
		name        string
		definition  Definition
		subscribers []Subscriber[orderPlaced]
		observers   []Observer
	}{
		{"missing event ID", Definition{Module: "example.com/shop/orders"}, nil, nil},
		{"missing event module", Definition{ID: "OrderPlaced"}, nil, nil},
		{
			"missing subscriber ID",
			eventDefinition(),
			[]Subscriber[orderPlaced]{{Module: "example.com/shop/listener", Handle: validSubscriber.Handle}},
			nil,
		},
		{
			"missing subscriber module",
			eventDefinition(),
			[]Subscriber[orderPlaced]{{ID: "listener", Handle: validSubscriber.Handle}},
			nil,
		},
		{
			"missing handler",
			eventDefinition(),
			[]Subscriber[orderPlaced]{{ID: "listener", Module: "example.com/shop/listener"}},
			nil,
		},
		{
			"duplicate subscriber",
			eventDefinition(),
			[]Subscriber[orderPlaced]{validSubscriber, validSubscriber},
			nil,
		},
		{"typed nil observer", eventDefinition(), nil, []Observer{typedNil}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewTopic(test.definition, test.subscribers, test.observers...); err == nil {
				t.Fatal("NewTopic() error = nil")
			}
		})
	}

	topic, err := NewTopic[orderPlaced](eventDefinition(), nil)
	if err != nil {
		t.Fatalf("NewTopic(no subscribers) error = %v", err)
	}
	if err := (*Topic[orderPlaced])(nil).Publish(context.Background(), orderPlaced{}); err == nil {
		t.Fatal("nil Topic.Publish() error = nil")
	}
	if err := topic.Publish(nilTestContext(), orderPlaced{}); err == nil {
		t.Fatal("Publish(nil context) error = nil")
	}
	if err := topic.Publish(context.Background(), orderPlaced{}); err != nil {
		t.Fatalf("Publish(no subscribers) error = %v", err)
	}
}

func TestTopicSupportsConcurrentPublish(t *testing.T) {
	t.Parallel()
	var delivered atomic.Int64
	topic, err := NewTopic(eventDefinition(), []Subscriber[int]{
		{
			ID:     "counter",
			Module: "example.com/shop/counter",
			Handle: func(_ context.Context, value int) error {
				delivered.Add(int64(value))
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTopic() error = %v", err)
	}
	const publishers = 64
	var wait sync.WaitGroup
	wait.Add(publishers)
	for range publishers {
		go func() {
			defer wait.Done()
			if publishErr := topic.Publish(context.Background(), 1); publishErr != nil {
				t.Errorf("Publish() error = %v", publishErr)
			}
		}()
	}
	wait.Wait()
	if got := delivered.Load(); got != publishers {
		t.Fatalf("delivered = %d, want %d", got, publishers)
	}
}

func TestEventObserversNestAndRetainContext(t *testing.T) {
	t.Parallel()
	var order []string
	first := observerFunc(func(ctx context.Context, _ Interaction) (context.Context, func(Result)) {
		order = append(order, "begin-first")
		return context.WithValue(ctx, observerContextKey{}, "observed"), func(Result) {
			order = append(order, "end-first")
		}
	})
	second := observerFunc(func(ctx context.Context, _ Interaction) (context.Context, func(Result)) {
		if ctx.Value(observerContextKey{}) != "observed" {
			t.Error("second observer did not receive first observer context")
		}
		order = append(order, "begin-second")
		return nil, func(Result) {
			order = append(order, "end-second")
		}
	})
	third := observerFunc(func(context.Context, Interaction) (context.Context, func(Result)) {
		order = append(order, "begin-third")
		return nil, nil
	})
	topic, err := NewTopic(eventDefinition(), []Subscriber[orderPlaced]{
		{
			ID:     "listener",
			Module: "example.com/shop/listener",
			Handle: func(context.Context, orderPlaced) error { return nil },
		},
	}, first, second, third)
	if err != nil {
		t.Fatalf("NewTopic() error = %v", err)
	}
	if err := topic.Publish(context.Background(), orderPlaced{}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	want := []string{"begin-first", "begin-second", "begin-third", "end-second", "end-first"}
	if !slices.Equal(order, want) {
		t.Fatalf("observer order = %v, want %v", order, want)
	}
}

func eventDefinition() Definition {
	return Definition{ID: "orders.OrderPlaced", Module: "example.com/shop/orders"}
}

func nilTestContext() context.Context {
	return nil
}

func recoverPublish(run func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	run()
	return nil
}

type observerContextKey struct{}

type observerFunc func(context.Context, Interaction) (context.Context, func(Result))

func (observe observerFunc) BeginEvent(
	ctx context.Context,
	interaction Interaction,
) (context.Context, func(Result)) {
	return observe(ctx, interaction)
}

type recordingObserver struct {
	mu      sync.Mutex
	results []Result
}

func (observer *recordingObserver) BeginEvent(
	ctx context.Context,
	_ Interaction,
) (context.Context, func(Result)) {
	return context.WithValue(ctx, observerContextKey{}, "observed"), func(result Result) {
		observer.mu.Lock()
		observer.results = append(observer.results, result)
		observer.mu.Unlock()
	}
}

func (observer *recordingObserver) resultsSnapshot() []Result {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	return append([]Result(nil), observer.results...)
}

func (observer *recordingObserver) onlyResult(t *testing.T) Result {
	t.Helper()
	results := observer.resultsSnapshot()
	if len(results) != 1 {
		t.Fatalf("results = %#v, want one", results)
	}
	return results[0]
}
