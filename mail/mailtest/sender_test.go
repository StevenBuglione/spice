package mailtest

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	spicemail "github.com/spice-framework/spice/mail"
)

func TestSenderRecordsInspectableImmutableDelivery(t *testing.T) {
	t.Parallel()
	message := testMessage(t)
	var sender *Sender
	var observed []Observation
	sender, err := New(Config{
		Capacity: 2,
		Observer: func(_ context.Context, observation Observation) {
			observed = append(observed, observation)
			if len(sender.Attempts()) != len(observed) {
				t.Fatal("observer ran before attempt became inspectable")
			}
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := sender.Send(context.Background(), message); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if sender.AttemptCount() != 1 {
		t.Fatalf("AttemptCount() = %d", sender.AttemptCount())
	}
	attempts := sender.Attempts()
	if len(attempts) != 1 ||
		attempts[0].Number() != 1 ||
		attempts[0].Outcome() != OutcomeDelivered ||
		attempts[0].Error() != nil {
		t.Fatalf("Attempts() = %#v", attempts)
	}
	snapshot := attempts[0].Message()
	assertSnapshot(t, snapshot)
	messages := sender.Messages()
	if len(messages) != 1 {
		t.Fatalf("Messages() = %#v", messages)
	}
	assertSnapshot(t, messages[0])
	if !slices.Equal(sender.Observations(), observed) ||
		len(observed) != 1 ||
		observed[0] != (Observation{
			Attempt:   1,
			MessageID: "order-41@example.com",
			Outcome:   OutcomeDelivered,
		}) {
		t.Fatalf("observations = %#v, callback = %#v", sender.Observations(), observed)
	}

	mutable := attempts[0].message
	mutable.recipients[0] = "changed@example.com"
	mutable.content[0] = 'X'
	mutable.attachments[0].data[0] = 'X'
	fresh := sender.Attempts()[0].Message()
	assertSnapshot(t, fresh)
}

func TestSenderInjectsDeterministicFailuresAndFiltersMessages(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("temporary test transport failure")
	failures := []error{sentinel, nil}
	sender, err := New(Config{
		Capacity: 2,
		Failures: failures,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	failures[0] = nil
	message := testMessage(t)
	if err := sender.Send(context.Background(), message); !errors.Is(err, sentinel) {
		t.Fatalf("first Send() error = %v, want sentinel", err)
	}
	if err := sender.Send(context.Background(), message); err != nil {
		t.Fatalf("second Send() error = %v", err)
	}

	attempts := sender.Attempts()
	if len(attempts) != 2 ||
		attempts[0].Outcome() != OutcomeFailed ||
		!errors.Is(attempts[0].Error(), sentinel) ||
		attempts[1].Outcome() != OutcomeDelivered {
		t.Fatalf("Attempts() = %#v", attempts)
	}
	if messages := sender.Messages(); len(messages) != 1 {
		t.Fatalf("Messages() = %#v", messages)
	}
	wantOutcomes := []Outcome{OutcomeFailed, OutcomeDelivered}
	observations := sender.Observations()
	for index, want := range wantOutcomes {
		if observations[index].Outcome != want {
			t.Fatalf("Observations()[%d] = %#v", index, observations[index])
		}
	}
}

func TestSenderFailsExplicitlyAtCapacity(t *testing.T) {
	t.Parallel()
	var observed []Observation
	sender, err := New(Config{
		Capacity: 1,
		Observer: func(_ context.Context, observation Observation) {
			observed = append(observed, observation)
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	message := testMessage(t)
	if sendErr := sender.Send(context.Background(), message); sendErr != nil {
		t.Fatalf("first Send() error = %v", sendErr)
	}
	err = sender.Send(context.Background(), message)
	if !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("second Send() error = %v", err)
	}
	var capacityErr *CapacityError
	if !errors.As(err, &capacityErr) ||
		capacityErr.Capacity != 1 ||
		capacityErr.Attempt != 2 {
		t.Fatalf("capacity error = %#v", capacityErr)
	}
	if sender.AttemptCount() != 2 ||
		len(sender.Attempts()) != 1 ||
		len(sender.Observations()) != 1 ||
		len(observed) != 2 ||
		observed[1].Outcome != OutcomeRejected {
		t.Fatalf(
			"count=%d attempts=%d observations=%#v callbacks=%#v",
			sender.AttemptCount(),
			len(sender.Attempts()),
			sender.Observations(),
			observed,
		)
	}
}

func TestSenderHonorsCancellationAndRejectsInvalidInputs(t *testing.T) {
	t.Parallel()
	sender, err := New(Config{Capacity: 1})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sender.Send(ctx, testMessage(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("Send(canceled) error = %v", err)
	}
	if sender.AttemptCount() != 0 {
		t.Fatalf("AttemptCount() = %d after pre-cancellation", sender.AttemptCount())
	}
	lateCancellation := &cancelAfterSnapshotContext{}
	if err := sender.Send(
		lateCancellation,
		testMessage(t),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("Send(late cancellation) error = %v", err)
	}
	attempts := sender.Attempts()
	if len(attempts) != 1 ||
		attempts[0].Outcome() != OutcomeCanceled ||
		len(sender.Messages()) != 0 {
		t.Fatalf(
			"late-canceled attempts=%#v messages=%#v",
			attempts,
			sender.Messages(),
		)
	}
	if err := sender.Send(nilContext(), testMessage(t)); err == nil {
		t.Fatal("Send(nil) error = nil")
	}
	if err := sender.Send(context.Background(), spicemail.Message{}); !errors.Is(
		err,
		ErrInvalidMessage,
	) {
		t.Fatalf("Send(zero message) error = %v", err)
	}
}

func TestNewRejectsUnboundedConfiguration(t *testing.T) {
	t.Parallel()
	for _, config := range []Config{
		{},
		{Capacity: -1},
		{Capacity: MaxCapacity + 1},
		{Capacity: 1, Failures: []error{nil, nil}},
	} {
		if sender, err := New(config); err == nil || sender != nil {
			t.Fatalf("New(%#v) = %#v, %v", config, sender, err)
		}
	}
}

func TestSenderIsConcurrencySafeAndBounded(t *testing.T) {
	t.Parallel()
	const count = 128
	sender, err := New(Config{Capacity: count})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	message := testMessage(t)
	errorsChannel := make(chan error, count)
	var wait sync.WaitGroup
	for range count {
		wait.Go(func() {
			errorsChannel <- sender.Send(context.Background(), message)
		})
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent Send() error = %v", err)
		}
	}
	attempts := sender.Attempts()
	if len(attempts) != count ||
		len(sender.Messages()) != count ||
		len(sender.Observations()) != count {
		t.Fatalf(
			"attempts=%d messages=%d observations=%d",
			len(attempts),
			len(sender.Messages()),
			len(sender.Observations()),
		)
	}
	wantNumber := uint64(1)
	for index, attempt := range attempts {
		if attempt.Number() != wantNumber {
			t.Fatalf("attempt %d number = %d", index, attempt.Number())
		}
		wantNumber++
	}
}

func FuzzSnapshotMIME(f *testing.F) {
	message := testMessage(f)
	f.Add(message.Bytes())
	f.Add([]byte("Subject: test\r\n\r\nbody"))
	f.Fuzz(func(t *testing.T, content []byte) {
		if len(content) > 1<<20 {
			t.Skip()
		}
		if _, err := snapshotMIME(
			"fuzz@example.com",
			"sender@example.com",
			[]string{"recipient@example.com"},
			content,
		); err != nil {
			return
		}
	})
}

func BenchmarkSenderSend(b *testing.B) {
	message := benchmarkMessage(b)
	b.ReportAllocs()
	for b.Loop() {
		sender, err := New(Config{Capacity: 1})
		if err != nil {
			b.Fatal(err)
		}
		if err := sender.Send(context.Background(), message); err != nil {
			b.Fatal(err)
		}
	}
}

func assertSnapshot(t *testing.T, snapshot Snapshot) {
	t.Helper()
	if snapshot.ID() != "order-41@example.com" ||
		snapshot.EnvelopeFrom() != "orders@example.com" ||
		!slices.Equal(snapshot.Recipients(), []string{
			"customer@example.com",
			"audit@example.com",
			"blind@example.com",
		}) ||
		snapshot.Subject() != "Order 41 is ready" ||
		snapshot.TextBody() != "plain\r\nbody" ||
		snapshot.HTMLBody() != "<p>HTML body</p>" {
		t.Fatalf(
			"snapshot id=%q from=%q recipients=%v subject=%q text=%q html=%q",
			snapshot.ID(),
			snapshot.EnvelopeFrom(),
			snapshot.Recipients(),
			snapshot.Subject(),
			snapshot.TextBody(),
			snapshot.HTMLBody(),
		)
	}
	attachments := snapshot.Attachments()
	if len(attachments) != 1 ||
		attachments[0].Filename() != "receipt.txt" ||
		attachments[0].ContentType() != "text/plain; charset=utf-8" ||
		string(attachments[0].Bytes()) != "receipt" {
		t.Fatalf("attachments = %#v", attachments)
	}
	if len(snapshot.Bytes()) == 0 {
		t.Fatal("snapshot MIME is empty")
	}
}

func testMessage(tb testing.TB) spicemail.Message {
	tb.Helper()
	return buildMessage(tb, "order-41@example.com")
}

func benchmarkMessage(tb testing.TB) spicemail.Message {
	tb.Helper()
	return buildMessage(tb, "benchmark@example.com")
}

func buildMessage(tb testing.TB, id string) spicemail.Message {
	tb.Helper()
	message, err := spicemail.NewMessage(spicemail.MessageSpec{
		ID:       id,
		Date:     time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC),
		From:     "Orders <orders@example.com>",
		To:       []string{"customer@example.com"},
		Cc:       []string{"audit@example.com"},
		Bcc:      []string{"blind@example.com"},
		Subject:  "Order 41 is ready",
		TextBody: "plain\nbody",
		HTMLBody: "<p>HTML body</p>",
		Attachments: []spicemail.AttachmentSpec{{
			Filename:    "receipt.txt",
			ContentType: "text/plain; charset=utf-8",
			Data:        []byte("receipt"),
		}},
	})
	if err != nil {
		tb.Fatalf("NewMessage() error = %v", err)
	}
	return message
}

type cancelAfterSnapshotContext struct {
	errCalls int
}

func (ctx *cancelAfterSnapshotContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (ctx *cancelAfterSnapshotContext) Done() <-chan struct{} {
	return nil
}

func (ctx *cancelAfterSnapshotContext) Err() error {
	ctx.errCalls++
	if ctx.errCalls > 1 {
		return context.Canceled
	}
	return nil
}

func (ctx *cancelAfterSnapshotContext) Value(any) any {
	return nil
}

func nilContext() context.Context {
	return nil
}
