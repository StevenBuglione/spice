package messaging

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewMessageFreezesAndNormalizes(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"order":"1"}`)
	headers := []Header{
		{Name: "Trace.ID", Value: "trace-1"},
		{Name: "correlation-id", Value: "correlation-1"},
	}
	occurredAt := time.Date(2026, 7, 29, 1, 2, 3, 0, time.FixedZone("test", 3600))
	message, err := NewMessage(MessageSpec{
		ID:          "message-1",
		Topic:       "orders.placed",
		Key:         "order-1",
		ContentType: "application/json; charset=UTF-8",
		Payload:     payload,
		Headers:     headers,
		OccurredAt:  occurredAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload[0] = '!'
	headers[0].Value = "changed"
	returnedPayload := message.Payload()
	returnedPayload[0] = '?'
	returnedHeaders := message.Headers()
	returnedHeaders[0].Value = "changed"

	if message.ID() != "message-1" ||
		message.Topic() != "orders.placed" ||
		message.Key() != "order-1" ||
		message.ContentType() != "application/json; charset=UTF-8" ||
		string(message.Payload()) != `{"order":"1"}` ||
		message.OccurredAt().Location() != time.UTC {
		t.Fatalf("message = %#v", message)
	}
	wantHeaders := []Header{
		{Name: "correlation-id", Value: "correlation-1"},
		{Name: "trace.id", Value: "trace-1"},
	}
	if got := message.Headers(); !equalHeaders(got, wantHeaders) {
		t.Fatalf("Headers() = %#v, want %#v", got, wantHeaders)
	}
}

func TestNewMessageRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	valid := MessageSpec{
		ID:          "message-1",
		Topic:       "orders.placed",
		ContentType: "application/json",
		Payload:     []byte("{}"),
		OccurredAt:  time.Now(),
	}
	tests := []struct {
		name   string
		mutate func(*MessageSpec)
	}{
		{name: "missing ID", mutate: func(spec *MessageSpec) { spec.ID = "" }},
		{name: "invalid topic", mutate: func(spec *MessageSpec) { spec.Topic = "orders placed" }},
		{name: "invalid key", mutate: func(spec *MessageSpec) { spec.Key = "/" }},
		{name: "invalid content type", mutate: func(spec *MessageSpec) { spec.ContentType = "json" }},
		{name: "missing payload", mutate: func(spec *MessageSpec) { spec.Payload = nil }},
		{name: "oversized payload", mutate: func(spec *MessageSpec) {
			spec.Payload = make([]byte, maxPayloadBytes+1)
		}},
		{name: "missing time", mutate: func(spec *MessageSpec) { spec.OccurredAt = time.Time{} }},
		{name: "duplicate header", mutate: func(spec *MessageSpec) {
			spec.Headers = []Header{{Name: "Trace", Value: "1"}, {Name: "trace", Value: "2"}}
		}},
		{name: "invalid header value", mutate: func(spec *MessageSpec) {
			spec.Headers = []Header{{Name: "trace", Value: "one\n two"}}
		}},
		{name: "too many headers", mutate: func(spec *MessageSpec) {
			spec.Headers = make([]Header, maxHeaderCount+1)
		}},
		{name: "oversized headers", mutate: func(spec *MessageSpec) {
			spec.Headers = []Header{{Name: "trace", Value: strings.Repeat("x", maxHeaderBytes)}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			spec := valid
			test.mutate(&spec)
			if _, err := NewMessage(spec); err == nil {
				t.Fatal("NewMessage() error = nil")
			}
		})
	}
}

func FuzzNewMessage(f *testing.F) {
	f.Add(
		"message-1",
		"orders.placed",
		"order-1",
		"application/json",
		"trace-id",
		"trace-1",
		[]byte("{}"),
	)
	f.Fuzz(func(
		t *testing.T,
		id string,
		topic string,
		key string,
		contentType string,
		headerName string,
		headerValue string,
		payload []byte,
	) {
		message, err := NewMessage(MessageSpec{
			ID:          id,
			Topic:       topic,
			Key:         key,
			ContentType: contentType,
			Headers: []Header{{
				Name:  headerName,
				Value: headerValue,
			}},
			Payload:    payload,
			OccurredAt: time.Unix(1, 0),
		})
		if err != nil {
			return
		}
		returned := message.Payload()
		if len(returned) != len(payload) {
			t.Fatalf(
				"Payload() length = %d, want %d",
				len(returned),
				len(payload),
			)
		}
		if len(returned) != 0 {
			returned[0] ^= 0xff
			if message.Payload()[0] != payload[0] {
				t.Fatal("Payload() did not return a defensive copy")
			}
		}
	})
}

func TestDeliveryHandleSettlesOutcomes(t *testing.T) {
	t.Parallel()
	handlerFailure := errors.New("handler failed")
	settlementFailure := errors.New("settlement failed")
	tests := []struct {
		name            string
		context         func() context.Context
		handler         Handler
		settlementError error
		wantDisposition Disposition
		wantCause       error
		wantError       error
	}{
		{
			name:            "success",
			context:         context.Background,
			handler:         func(context.Context, Message) error { return nil },
			wantDisposition: DispositionAcknowledge,
		},
		{
			name:            "handler failure",
			context:         context.Background,
			handler:         func(context.Context, Message) error { return handlerFailure },
			wantDisposition: DispositionRetry,
			wantCause:       handlerFailure,
			wantError:       handlerFailure,
		},
		{
			name:    "panic",
			context: context.Background,
			handler: func(context.Context, Message) error {
				panic("boom")
			},
			wantDisposition: DispositionRetry,
			wantCause:       ErrHandlerPanicked,
			wantError:       ErrHandlerPanicked,
		},
		{
			name: "canceled",
			context: func() context.Context {
				ctx, cancel := context.WithCancelCause(context.Background())
				cancel(context.Canceled)
				return ctx
			},
			handler:         func(context.Context, Message) error { return nil },
			wantDisposition: DispositionReject,
			wantCause:       context.Canceled,
			wantError:       context.Canceled,
		},
		{
			name:            "joined settlement failure",
			context:         context.Background,
			handler:         func(context.Context, Message) error { return handlerFailure },
			settlementError: settlementFailure,
			wantDisposition: DispositionRetry,
			wantCause:       handlerFailure,
			wantError:       settlementFailure,
		},
		{
			name:            "settlement panic",
			context:         context.Background,
			handler:         func(context.Context, Message) error { return nil },
			wantDisposition: DispositionAcknowledge,
			wantError:       ErrSettlementPanicked,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			settlement := &recordingSettlement{err: test.settlementError}
			if test.name == "settlement panic" {
				settlement.panics = true
			}
			delivery := testDelivery(t, settlement)
			err := delivery.Handle(test.context(), test.handler)
			if test.wantError == nil && err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if test.wantError != nil && !errors.Is(err, test.wantError) {
				t.Fatalf("Handle() error = %v, want %v", err, test.wantError)
			}
			if settlement.disposition != test.wantDisposition ||
				!errors.Is(settlement.cause, test.wantCause) {
				t.Fatalf(
					"settlement = (%q, %v), want (%q, %v)",
					settlement.disposition,
					settlement.cause,
					test.wantDisposition,
					test.wantCause,
				)
			}
		})
	}
}

func TestDeliveryHandleSettlesOnceAcrossCopies(t *testing.T) {
	t.Parallel()
	settlement := &countingSettlement{}
	delivery := testDelivery(t, settlement)
	const callers = 32
	var handlers atomic.Int32
	var failures atomic.Int32
	var wait sync.WaitGroup
	wait.Add(callers)
	for range callers {
		go func(copied Delivery) {
			defer wait.Done()
			err := copied.Handle(
				context.Background(),
				func(context.Context, Message) error {
					handlers.Add(1)
					return nil
				},
			)
			if err != nil {
				failures.Add(1)
			}
		}(delivery)
	}
	wait.Wait()
	if handlers.Load() != 1 ||
		settlement.calls.Load() != 1 ||
		failures.Load() != callers-1 {
		t.Fatalf(
			"handlers=%d settlements=%d failures=%d",
			handlers.Load(),
			settlement.calls.Load(),
			failures.Load(),
		)
	}
}

func TestDeliveryRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	message := testMessage(t)
	validMetadata := DeliveryMetadata{
		Consumer:   "orders",
		Attempt:    1,
		ReceivedAt: time.Now(),
	}
	settlement := &recordingSettlement{}
	if _, err := NewDelivery(Message{}, validMetadata, settlement); err == nil {
		t.Fatal("NewDelivery(invalid message) error = nil")
	}
	metadata := validMetadata
	metadata.Consumer = ""
	if _, err := NewDelivery(message, metadata, settlement); err == nil {
		t.Fatal("NewDelivery(missing consumer) error = nil")
	}
	metadata = validMetadata
	metadata.Attempt = 0
	if _, err := NewDelivery(message, metadata, settlement); err == nil {
		t.Fatal("NewDelivery(invalid attempt) error = nil")
	}
	metadata = validMetadata
	metadata.ReceivedAt = time.Time{}
	if _, err := NewDelivery(message, metadata, settlement); err == nil {
		t.Fatal("NewDelivery(missing time) error = nil")
	}
	if _, err := NewDelivery(message, validMetadata, nil); err == nil {
		t.Fatal("NewDelivery(nil settlement) error = nil")
	}
	var typedNil *recordingSettlement
	if _, err := NewDelivery(message, validMetadata, typedNil); err == nil {
		t.Fatal("NewDelivery(typed nil settlement) error = nil")
	}

	delivery := testDelivery(t, settlement)
	if err := delivery.Handle(
		nilContext(),
		func(context.Context, Message) error { return nil },
	); err == nil {
		t.Fatal("Handle(nil context) error = nil")
	}
	if err := delivery.Handle(context.Background(), nil); err == nil {
		t.Fatal("Handle(nil handler) error = nil")
	}
	if err := (Delivery{}).Handle(context.Background(), func(context.Context, Message) error {
		return nil
	}); err == nil {
		t.Fatal("Handle(invalid delivery) error = nil")
	}
	if err := delivery.Handle(
		context.Background(),
		func(context.Context, Message) error { return nil },
	); err != nil {
		t.Fatalf("first Handle() error = %v", err)
	}
	if err := delivery.Handle(
		context.Background(),
		func(context.Context, Message) error { return nil },
	); err == nil || !strings.Contains(err.Error(), "already settled") {
		t.Fatalf("second Handle() error = %v", err)
	}
}

func testDelivery(t *testing.T, settlement Settlement) Delivery {
	t.Helper()
	delivery, err := NewDelivery(testMessage(t), DeliveryMetadata{
		Consumer:   "orders",
		Attempt:    1,
		ReceivedAt: time.Now(),
	}, settlement)
	if err != nil {
		t.Fatal(err)
	}
	return delivery
}

func testMessage(t *testing.T) Message {
	t.Helper()
	message, err := NewMessage(MessageSpec{
		ID:          "message-1",
		Topic:       "orders.placed",
		ContentType: "application/json",
		Payload:     []byte("{}"),
		OccurredAt:  time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return message
}

type recordingSettlement struct {
	disposition Disposition
	cause       error
	err         error
	panics      bool
}

func (settlement *recordingSettlement) Settle(
	_ context.Context,
	disposition Disposition,
	cause error,
) error {
	settlement.disposition = disposition
	settlement.cause = cause
	if settlement.panics {
		panic("secret settlement failure")
	}
	return settlement.err
}

type countingSettlement struct {
	calls atomic.Int32
}

func (settlement *countingSettlement) Settle(
	context.Context,
	Disposition,
	error,
) error {
	settlement.calls.Add(1)
	return nil
}

func equalHeaders(left, right []Header) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func nilContext() context.Context {
	return nil
}
