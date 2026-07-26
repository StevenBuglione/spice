package batch

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

var errSQLTest = errors.New("database failed")

func TestSQLStoreExecutesExactAtomicProtocol(t *testing.T) {
	t.Parallel()

	now := time.Date(
		2026,
		time.July,
		26,
		12,
		0,
		0,
		0,
		time.FixedZone("test", -4*60*60),
	)
	state := &batchSQLState{
		row:      []driver.Value{"resumed", int64(2), []byte(`["extract"]`)},
		affected: 1,
	}
	store := newBatchSQLStore(t, state, func() time.Time { return now })
	request := testBeginRequest("private-instance")
	attempt, err := store.Begin(context.Background(), request)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if attempt.Number() != 2 ||
		!slices.Equal(attempt.CompletedSteps(), []string{"extract"}) {
		t.Fatalf("attempt = %#v", attempt)
	}
	if err := store.Checkpoint(
		context.Background(),
		attempt,
		"load",
	); err != nil {
		t.Fatalf("Checkpoint() error = %v", err)
	}
	if err := store.Complete(context.Background(), attempt); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if err := store.Fail(context.Background(), Failure{
		Attempt: attempt,
		Step:    "load",
		Kind:    FailureCanceled,
	}); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}

	queries, execs := state.calls()
	if len(queries) != 1 || queries[0].statement != "BEGIN" {
		t.Fatalf("queries = %#v", queries)
	}
	begin := queries[0].arguments
	if len(begin) != 6 {
		t.Fatalf("begin arguments = %#v", begin)
	}
	encodedSteps, stepsOK := begin[3].([]byte)
	currentTime, currentTimeOK := begin[4].(time.Time)
	leaseExpiry, leaseExpiryOK := begin[5].(time.Time)
	if begin[0] != "orders.import" ||
		begin[1] != "example.com/shop/orders" ||
		begin[2] != "private-instance" ||
		!stepsOK ||
		string(encodedSteps) != `["extract","load"]` ||
		!currentTimeOK ||
		!currentTime.Equal(now.UTC()) ||
		!leaseExpiryOK ||
		!leaseExpiry.Equal(now.UTC().Add(time.Minute)) {
		t.Fatalf("begin arguments = %#v", begin)
	}
	if got := batchSQLCallNames(execs); !slices.Equal(
		got,
		[]string{"CHECKPOINT", "COMPLETE", "FAIL"},
	) {
		t.Fatalf("exec statements = %v", got)
	}
	if len(execs[0].arguments) != 7 {
		t.Fatalf("checkpoint arguments = %#v", execs[0].arguments)
	}
	checkpointExpiry, checkpointExpiryOK := execs[0].arguments[6].(time.Time)
	if execs[0].arguments[4] != "load" ||
		!checkpointExpiryOK ||
		!checkpointExpiry.Equal(now.UTC().Add(time.Minute)) {
		t.Fatalf("checkpoint arguments = %#v", execs[0].arguments)
	}
	if len(execs[1].arguments) != 5 ||
		len(execs[2].arguments) != 7 ||
		execs[2].arguments[5] != string(FailureCanceled) {
		t.Fatalf("transition arguments = %#v", execs)
	}
}

func TestSQLStoreValidatesConstructionAndRequests(t *testing.T) {
	t.Parallel()

	statements := batchSQLStatements()
	options := SQLStoreOptions{AttemptLease: time.Minute}
	if _, err := NewSQLStore(nil, statements, options); err == nil {
		t.Fatal("NewSQLStore(nil) unexpectedly succeeded")
	}
	for index, invalid := range []SQLStatements{
		{Checkpoint: "q", Complete: "q", Fail: "q"},
		{Begin: "q", Complete: "q", Fail: "q"},
		{Begin: "q", Checkpoint: "q", Fail: "q"},
		{Begin: "q", Checkpoint: "q", Complete: "q"},
	} {
		database := newBatchSQLDatabase(t, &batchSQLState{})
		if _, err := NewSQLStore(database, invalid, options); err == nil {
			t.Fatalf("NewSQLStore(invalid %d) unexpectedly succeeded", index)
		}
	}
	database := newBatchSQLDatabase(t, &batchSQLState{})
	if _, err := NewSQLStore(database, statements, SQLStoreOptions{}); err == nil {
		t.Fatal("NewSQLStore(zero lease) unexpectedly succeeded")
	}
	if _, err := NewSQLStore(database, statements, SQLStoreOptions{
		AttemptLease: maxSQLAttemptLease + 1,
	}); err == nil {
		t.Fatal("NewSQLStore(large lease) unexpectedly succeeded")
	}

	store := newBatchSQLStore(t, &batchSQLState{}, time.Now)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Begin(
		canceled,
		testBeginRequest("instance"),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("Begin(canceled) error = %v", err)
	}
	if _, err := store.Begin(nilBatchContext(), BeginRequest{}); err == nil {
		t.Fatal("Begin(nil context) unexpectedly succeeded")
	}
	if _, err := (*SQLStore)(nil).Begin(
		context.Background(),
		BeginRequest{},
	); err == nil {
		t.Fatal("nil Begin() unexpectedly succeeded")
	}
	if _, err := store.Begin(
		context.Background(),
		BeginRequest{},
	); err == nil {
		t.Fatal("Begin(invalid request) unexpectedly succeeded")
	}
	if err := store.Checkpoint(
		context.Background(),
		Attempt{},
		"step",
	); err == nil {
		t.Fatal("Checkpoint(invalid attempt) unexpectedly succeeded")
	}
	largeAttempt := mustAttempt(t, AttemptSpec{
		Definition: testDefinition(),
		Instance:   "instance",
		Number:     maxSQLAttemptNumber + 1,
	})
	if err := store.Complete(
		context.Background(),
		largeAttempt,
	); err == nil {
		t.Fatal("Complete(large attempt) unexpectedly succeeded")
	}
	if err := store.Fail(context.Background(), Failure{}); err == nil {
		t.Fatal("Fail(invalid failure) unexpectedly succeeded")
	}

	zeroClockStore := newBatchSQLStore(
		t,
		&batchSQLState{},
		func() time.Time { return time.Time{} },
	)
	if _, err := zeroClockStore.Begin(
		context.Background(),
		testBeginRequest("instance"),
	); err == nil {
		t.Fatal("Begin(zero clock) unexpectedly succeeded")
	}
}

func TestSQLStoreRejectsUnsafeTransitionResults(t *testing.T) {
	t.Parallel()

	attempt := mustAttempt(t, AttemptSpec{
		Definition: testDefinition(),
		Instance:   "secret-instance",
		Number:     3,
	})
	tests := []struct {
		name      string
		state     *batchSQLState
		wantStale bool
	}{
		{name: "zero rows", state: &batchSQLState{}, wantStale: true},
		{name: "multiple rows", state: &batchSQLState{affected: 2}},
		{name: "execute error", state: &batchSQLState{execErr: errSQLTest}},
		{name: "result error", state: &batchSQLState{
			affected:  1,
			resultErr: errSQLTest,
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := newBatchSQLStore(t, test.state, time.Now)
			err := store.Checkpoint(context.Background(), attempt, "extract")
			if test.wantStale && !errors.Is(err, ErrStaleAttempt) {
				t.Fatalf("Checkpoint() error = %v, want ErrStaleAttempt", err)
			}
			if !test.wantStale && err == nil {
				t.Fatal("Checkpoint() unexpectedly succeeded")
			}
			if err != nil && strings.Contains(err.Error(), "secret-instance") {
				t.Fatalf("Checkpoint() exposed instance: %v", err)
			}
		})
	}
}

type batchSQLCall struct {
	statement string
	arguments []any
}

type batchSQLState struct {
	mu        sync.Mutex
	row       []driver.Value
	queries   []batchSQLCall
	execs     []batchSQLCall
	queryErr  error
	execErr   error
	resultErr error
	affected  int64
}

func (state *batchSQLState) calls() ([]batchSQLCall, []batchSQLCall) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return slices.Clone(state.queries), slices.Clone(state.execs)
}

type batchSQLConnector struct{ state *batchSQLState }

func (connector *batchSQLConnector) Connect(context.Context) (driver.Conn, error) {
	return &batchSQLConnection{state: connector.state}, nil
}

func (*batchSQLConnector) Driver() driver.Driver { return batchSQLDriver{} }

type batchSQLDriver struct{}

func (batchSQLDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use connector")
}

type batchSQLConnection struct{ state *batchSQLState }

func (*batchSQLConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}

func (*batchSQLConnection) Close() error { return nil }
func (*batchSQLConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}

func (connection *batchSQLConnection) QueryContext(
	_ context.Context,
	statement string,
	arguments []driver.NamedValue,
) (driver.Rows, error) {
	connection.state.mu.Lock()
	defer connection.state.mu.Unlock()
	connection.state.queries = append(connection.state.queries, batchSQLCall{
		statement: statement,
		arguments: batchSQLArguments(arguments),
	})
	if connection.state.queryErr != nil {
		return nil, connection.state.queryErr
	}
	rows := [][]driver.Value{}
	if connection.state.row != nil {
		rows = append(rows, slices.Clone(connection.state.row))
	}
	return &batchSQLRows{rows: rows}, nil
}

func (connection *batchSQLConnection) ExecContext(
	_ context.Context,
	statement string,
	arguments []driver.NamedValue,
) (driver.Result, error) {
	connection.state.mu.Lock()
	defer connection.state.mu.Unlock()
	connection.state.execs = append(connection.state.execs, batchSQLCall{
		statement: statement,
		arguments: batchSQLArguments(arguments),
	})
	if connection.state.execErr != nil {
		return nil, connection.state.execErr
	}
	return batchSQLResult{
		affected: connection.state.affected,
		err:      connection.state.resultErr,
	}, nil
}

type batchSQLRows struct {
	rows  [][]driver.Value
	index int
}

func (*batchSQLRows) Columns() []string {
	return []string{"outcome", "attempt", "completed_steps"}
}

func (*batchSQLRows) Close() error { return nil }

func (rows *batchSQLRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.rows) {
		return io.EOF
	}
	copy(destination, rows.rows[rows.index])
	rows.index++
	return nil
}

type batchSQLResult struct {
	affected int64
	err      error
}

func (batchSQLResult) LastInsertId() (int64, error) {
	return 0, errors.New("unsupported")
}

func (result batchSQLResult) RowsAffected() (int64, error) {
	return result.affected, result.err
}

func newBatchSQLStore(
	t *testing.T,
	state *batchSQLState,
	clock func() time.Time,
) *SQLStore {
	t.Helper()
	database := newBatchSQLDatabase(t, state)
	store, err := NewSQLStore(
		database,
		batchSQLStatements(),
		SQLStoreOptions{AttemptLease: time.Minute, Clock: clock},
	)
	if err != nil {
		t.Fatalf("NewSQLStore() error = %v", err)
	}
	return store
}

func newBatchSQLDatabase(
	t *testing.T,
	state *batchSQLState,
) *sql.DB {
	t.Helper()
	database := sql.OpenDB(&batchSQLConnector{state: state})
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("database Close() error = %v", err)
		}
	})
	return database
}

func batchSQLStatements() SQLStatements {
	return SQLStatements{
		Begin:      "BEGIN",
		Checkpoint: "CHECKPOINT",
		Complete:   "COMPLETE",
		Fail:       "FAIL",
	}
}

func batchSQLArguments(arguments []driver.NamedValue) []any {
	result := make([]any, len(arguments))
	for index, argument := range arguments {
		result[index] = argument.Value
	}
	return result
}

func batchSQLCallNames(calls []batchSQLCall) []string {
	result := make([]string, len(calls))
	for index, call := range calls {
		result[index] = call.statement
	}
	return result
}

func nilBatchContext() context.Context {
	return nil
}
