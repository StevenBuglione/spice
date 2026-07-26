package batch

import (
	"context"
	"database/sql/driver"
	"errors"
	"slices"
	"testing"
	"time"
)

func TestSQLStoreInterpretsBeginOutcomesFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		row            []driver.Value
		wantError      error
		wantComplete   bool
		wantCompleted  []string
		expectAnyError bool
	}{
		{
			name:          "started",
			row:           []driver.Value{"started", int64(1), []byte(`[]`)},
			wantCompleted: []string{},
		},
		{
			name:          "resumed",
			row:           []driver.Value{"resumed", int64(2), []byte(`["extract"]`)},
			wantCompleted: []string{"extract"},
		},
		{
			name:          "complete",
			row:           []driver.Value{"complete", int64(2), []byte(`["extract","load"]`)},
			wantComplete:  true,
			wantCompleted: []string{"extract", "load"},
		},
		{
			name:      "running",
			row:       []driver.Value{"running", int64(1), []byte(`[]`)},
			wantError: ErrAlreadyRunning,
		},
		{
			name:      "changed",
			row:       []driver.Value{"changed", int64(1), []byte(`[]`)},
			wantError: ErrDefinitionChanged,
		},
		{
			name:           "overflow",
			row:            []driver.Value{"overflow", int64(1), []byte(`[]`)},
			expectAnyError: true,
		},
		{
			name:           "unknown",
			row:            []driver.Value{"mystery", int64(1), []byte(`[]`)},
			expectAnyError: true,
		},
		{
			name:           "zero attempt",
			row:            []driver.Value{"resumed", int64(0), []byte(`[]`)},
			expectAnyError: true,
		},
		{
			name:           "invalid JSON",
			row:            []driver.Value{"resumed", int64(2), []byte(`{`)},
			expectAnyError: true,
		},
		{
			name:           "non-prefix",
			row:            []driver.Value{"resumed", int64(2), []byte(`["load"]`)},
			expectAnyError: true,
		},
		{
			name:           "started with checkpoint",
			row:            []driver.Value{"started", int64(1), []byte(`["extract"]`)},
			expectAnyError: true,
		},
		{
			name:           "complete with pending step",
			row:            []driver.Value{"complete", int64(2), []byte(`["extract"]`)},
			expectAnyError: true,
		},
		{name: "no row", expectAnyError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			state := &batchSQLState{row: test.row, affected: 1}
			store := newBatchSQLStore(t, state, time.Now)
			attempt, err := store.Begin(
				context.Background(),
				testBeginRequest("instance"),
			)
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf("Begin() error = %v, want %v", err, test.wantError)
				}
				return
			}
			if test.expectAnyError {
				if err == nil {
					t.Fatal("Begin() unexpectedly succeeded")
				}
				return
			}
			if err != nil {
				t.Fatalf("Begin() error = %v", err)
			}
			if attempt.Complete() != test.wantComplete ||
				!slices.Equal(attempt.CompletedSteps(), test.wantCompleted) {
				t.Fatalf("attempt = %#v", attempt)
			}
		})
	}
}
