package outbox

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/StevenBuglione/spice/data"
)

func TestSQLStoreExecutesExactProtocol(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	message := testMessage(t, "message-1", now)
	state := &sqlTestState{
		rows: [][]driver.Value{
			testSQLRow("message-1", "lease-1", now, 1),
			testSQLRow("message-2", "lease-2", now.Add(time.Second), 2),
		},
		mutatePayload: true,
	}
	store, db := newSQLTestStore(t, state)
	if err := store.Enqueue(context.Background(), db, message); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	deliveries, err := store.Claim(context.Background(), ClaimRequest{
		Owner: "worker-1",
		Now:   now,
		Lease: time.Minute,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if len(deliveries) != 2 ||
		deliveries[0].Message().ID() != "message-1" ||
		deliveries[1].Attempt() != 2 {
		t.Fatalf("deliveries = %#v", deliveries)
	}
	if err := store.Complete(context.Background(), Completion{
		Owner: "worker-1", Receipt: "lease-1",
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if err := store.Release(context.Background(), Release{
		Owner:       "worker-1",
		Receipt:     "lease-2",
		AvailableAt: now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if string(message.Payload()) != `{"order":"1"}` {
		t.Fatalf("message payload was mutated: %q", message.Payload())
	}

	execs, queries := state.calls()
	if got, want := callNames(execs), []string{"INSERT", "COMPLETE", "RELEASE"}; !slices.Equal(got, want) {
		t.Fatalf("exec statements = %v, want %v", got, want)
	}
	if len(execs[0].arguments) != 6 {
		t.Fatalf("insert arguments = %#v", execs[0].arguments)
	}
	insertedAt, insertedAtOK := execs[0].arguments[5].(time.Time)
	if execs[0].arguments[0] != "message-1" ||
		execs[0].arguments[1] != "orders.OrderPlaced" ||
		execs[0].arguments[2] != "example.com/shop/orders" ||
		execs[0].arguments[3] != "application/json" ||
		!insertedAtOK ||
		!insertedAt.Equal(now) {
		t.Fatalf("insert arguments = %#v", execs[0].arguments)
	}
	if len(queries) != 1 {
		t.Fatalf("query calls = %#v", queries)
	}
	if len(queries[0].arguments) != 4 {
		t.Fatalf("claim arguments = %#v", queries[0].arguments)
	}
	claimNow, claimNowOK := queries[0].arguments[1].(time.Time)
	leaseUntil, leaseUntilOK := queries[0].arguments[2].(time.Time)
	if queries[0].statement != "CLAIM" ||
		queries[0].arguments[0] != "worker-1" ||
		!claimNowOK ||
		!claimNow.Equal(now) ||
		!leaseUntilOK ||
		!leaseUntil.Equal(now.Add(time.Minute)) ||
		queries[0].arguments[3] != int64(10) {
		t.Fatalf("query calls = %#v", queries)
	}
}

func TestSQLStoreRejectsInvalidConstructionAndRequests(t *testing.T) {
	t.Parallel()
	state := &sqlTestState{}
	_, db := newSQLTestStore(t, state)
	valid := testStatements()
	if _, err := NewSQLStore(nil, valid); err == nil {
		t.Fatal("NewSQLStore(nil executor) error = nil")
	}
	for index, statements := range []SQLStatements{
		{Claim: "q", Complete: "q", Release: "q"},
		{Insert: "q", Complete: "q", Release: "q"},
		{Insert: "q", Claim: "q", Release: "q"},
		{Insert: "q", Claim: "q", Complete: "q"},
	} {
		if _, err := NewSQLStore(db, statements); err == nil {
			t.Fatalf("NewSQLStore(case %d) error = nil", index)
		}
	}
	store, err := NewSQLStore(db, valid)
	if err != nil {
		t.Fatalf("NewSQLStore() error = %v", err)
	}
	message := testMessage(t, "message-1", time.Now())
	if err := store.Enqueue(nilContext(), db, message); err == nil {
		t.Fatal("Enqueue(nil context) error = nil")
	}
	if err := store.Enqueue(context.Background(), nil, message); err == nil {
		t.Fatal("Enqueue(nil executor) error = nil")
	}
	if err := store.Enqueue(context.Background(), db, Message{}); err == nil {
		t.Fatal("Enqueue(invalid message) error = nil")
	}
	if err := (*SQLStore)(nil).Enqueue(context.Background(), db, message); err == nil {
		t.Fatal("nil Enqueue() error = nil")
	}
	invalidClaims := []ClaimRequest{
		{},
		{Owner: "worker", Lease: time.Second, Limit: 1},
		{Owner: "worker", Now: time.Now(), Limit: 1},
		{Owner: "worker", Now: time.Now(), Lease: maxFailureDelay + 1, Limit: 1},
		{Owner: "worker", Now: time.Now(), Lease: time.Second},
		{Owner: "worker", Now: time.Now(), Lease: time.Second, Limit: 1001},
	}
	for index, request := range invalidClaims {
		if _, err := store.Claim(context.Background(), request); err == nil {
			t.Fatalf("Claim(case %d) error = nil", index)
		}
	}
	if _, err := store.Claim(nilContext(), ClaimRequest{}); err == nil {
		t.Fatal("Claim(nil context) error = nil")
	}
	if _, err := (*SQLStore)(nil).Claim(context.Background(), ClaimRequest{}); err == nil {
		t.Fatal("nil Claim() error = nil")
	}
	if err := store.Complete(nilContext(), Completion{}); err == nil {
		t.Fatal("Complete(nil context) error = nil")
	}
	if err := store.Complete(context.Background(), Completion{}); err == nil {
		t.Fatal("Complete(invalid request) error = nil")
	}
	if err := (*SQLStore)(nil).Complete(context.Background(), Completion{}); err == nil {
		t.Fatal("nil Complete() error = nil")
	}
	if err := store.Release(nilContext(), Release{}); err == nil {
		t.Fatal("Release(nil context) error = nil")
	}
	if err := store.Release(context.Background(), Release{Owner: "worker", Receipt: "lease"}); err == nil {
		t.Fatal("Release(no time) error = nil")
	}
	if err := (*SQLStore)(nil).Release(context.Background(), Release{}); err == nil {
		t.Fatal("nil Release() error = nil")
	}
}

func TestSQLStoreReportsDatabaseAndRowFailures(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	tests := []struct {
		name      string
		state     *sqlTestState
		operation func(*SQLStore, *sql.DB) error
	}{
		{
			name:  "execute",
			state: &sqlTestState{execErr: errTest},
			operation: func(store *SQLStore, db *sql.DB) error {
				return store.Enqueue(context.Background(), db, testMessage(t, "message-1", now))
			},
		},
		{
			name:  "affected rows",
			state: &sqlTestState{affected: 2},
			operation: func(store *SQLStore, _ *sql.DB) error {
				return store.Complete(context.Background(), Completion{
					Owner: "worker", Receipt: "lease",
				})
			},
		},
		{
			name:  "affected rows error",
			state: &sqlTestState{resultErr: errTest},
			operation: func(store *SQLStore, _ *sql.DB) error {
				return store.Complete(context.Background(), Completion{
					Owner: "worker", Receipt: "lease",
				})
			},
		},
		{
			name:  "query",
			state: &sqlTestState{queryErr: errTest},
			operation: func(store *SQLStore, _ *sql.DB) error {
				_, err := store.Claim(context.Background(), validClaim(now))
				return err
			},
		},
		{
			name:  "scan",
			state: &sqlTestState{columns: []string{"id"}, rows: [][]driver.Value{{"message"}}},
			operation: func(store *SQLStore, _ *sql.DB) error {
				_, err := store.Claim(context.Background(), validClaim(now))
				return err
			},
		},
		{
			name:  "reconstruct",
			state: &sqlTestState{rows: [][]driver.Value{testSQLRow("", "lease", now, 1)}},
			operation: func(store *SQLStore, _ *sql.DB) error {
				_, err := store.Claim(context.Background(), validClaim(now))
				return err
			},
		},
		{
			name: "row iteration",
			state: &sqlTestState{
				rowsErr: errTest,
			},
			operation: func(store *SQLStore, _ *sql.DB) error {
				_, err := store.Claim(context.Background(), validClaim(now))
				return err
			},
		},
		{
			name: "order",
			state: &sqlTestState{rows: [][]driver.Value{
				testSQLRow("message-2", "lease-2", now.Add(time.Second), 1),
				testSQLRow("message-1", "lease-1", now, 1),
			}},
			operation: func(store *SQLStore, _ *sql.DB) error {
				_, err := store.Claim(context.Background(), validClaim(now))
				return err
			},
		},
	}
	for _, test := range tests {
		store, db := newSQLTestStore(t, test.state)
		if err := test.operation(store, db); err == nil {
			t.Fatalf("%s: operation error = nil", test.name)
		}
	}
}

type sqlCall struct {
	statement string
	arguments []any
}

type sqlTestState struct {
	mu            sync.Mutex
	execs         []sqlCall
	queries       []sqlCall
	columns       []string
	rows          [][]driver.Value
	execErr       error
	queryErr      error
	rowsErr       error
	resultErr     error
	affected      int64
	mutatePayload bool
}

func (state *sqlTestState) calls() ([]sqlCall, []sqlCall) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return append([]sqlCall(nil), state.execs...), append([]sqlCall(nil), state.queries...)
}

type sqlTestConnector struct {
	state *sqlTestState
}

func (connector *sqlTestConnector) Connect(context.Context) (driver.Conn, error) {
	return &sqlTestConnection{state: connector.state}, nil
}

func (*sqlTestConnector) Driver() driver.Driver {
	return sqlTestDriver{}
}

type sqlTestDriver struct{}

func (sqlTestDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use connector")
}

type sqlTestConnection struct {
	state *sqlTestState
}

func (*sqlTestConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}

func (*sqlTestConnection) Close() error {
	return nil
}

func (*sqlTestConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}

func (connection *sqlTestConnection) ExecContext(
	_ context.Context,
	statement string,
	arguments []driver.NamedValue,
) (driver.Result, error) {
	values := namedValues(arguments)
	connection.state.mu.Lock()
	connection.state.execs = append(connection.state.execs, sqlCall{
		statement: statement,
		arguments: values,
	})
	if connection.state.mutatePayload && statement == "INSERT" {
		if payload, ok := values[4].([]byte); ok && len(payload) != 0 {
			payload[0] = '!'
		}
	}
	err := connection.state.execErr
	resultErr := connection.state.resultErr
	affected := connection.state.affected
	connection.state.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		affected = 1
	}
	return sqlTestResult{affected: affected, err: resultErr}, nil
}

func (connection *sqlTestConnection) QueryContext(
	_ context.Context,
	statement string,
	arguments []driver.NamedValue,
) (driver.Rows, error) {
	connection.state.mu.Lock()
	connection.state.queries = append(connection.state.queries, sqlCall{
		statement: statement,
		arguments: namedValues(arguments),
	})
	queryErr := connection.state.queryErr
	columns := append([]string(nil), connection.state.columns...)
	rows := cloneDriverRows(connection.state.rows)
	rowsErr := connection.state.rowsErr
	connection.state.mu.Unlock()
	if queryErr != nil {
		return nil, queryErr
	}
	if len(columns) == 0 {
		columns = []string{
			"id", "topic", "module", "content_type",
			"payload", "occurred_at", "receipt", "attempt",
		}
	}
	return &sqlTestRows{columns: columns, rows: rows, finalErr: rowsErr}, nil
}

type sqlTestResult struct {
	affected int64
	err      error
}

func (result sqlTestResult) LastInsertId() (int64, error) {
	return 0, errors.New("unsupported")
}

func (result sqlTestResult) RowsAffected() (int64, error) {
	return result.affected, result.err
}

type sqlTestRows struct {
	columns  []string
	rows     [][]driver.Value
	index    int
	finalErr error
}

func (rows *sqlTestRows) Columns() []string {
	return rows.columns
}

func (*sqlTestRows) Close() error {
	return nil
}

func (rows *sqlTestRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.rows) {
		return rows.finalErrOrEOF()
	}
	copy(destination, rows.rows[rows.index])
	rows.index++
	return nil
}

func (rows *sqlTestRows) finalErrOrEOF() error {
	if rows.finalErr != nil {
		err := rows.finalErr
		rows.finalErr = nil
		return err
	}
	return io.EOF
}

func newSQLTestStore(t *testing.T, state *sqlTestState) (*SQLStore, *sql.DB) {
	t.Helper()
	db := sql.OpenDB(&sqlTestConnector{state: state})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("database Close() error = %v", err)
		}
	})
	store, err := NewSQLStore(db, testStatements())
	if err != nil {
		t.Fatalf("NewSQLStore() error = %v", err)
	}
	return store, db
}

func testStatements() SQLStatements {
	return SQLStatements{
		Insert: "INSERT", Claim: "CLAIM", Complete: "COMPLETE", Release: "RELEASE",
	}
}

func validClaim(now time.Time) ClaimRequest {
	return ClaimRequest{Owner: "worker", Now: now, Lease: time.Minute, Limit: 10}
}

func testSQLRow(id, receipt string, occurredAt time.Time, attempt int64) []driver.Value {
	return []driver.Value{
		id,
		"orders.OrderPlaced",
		"example.com/shop/orders",
		"application/json",
		[]byte(`{"order":"1"}`),
		occurredAt,
		receipt,
		attempt,
	}
}

func namedValues(arguments []driver.NamedValue) []any {
	values := make([]any, len(arguments))
	for index, argument := range arguments {
		values[index] = argument.Value
	}
	return values
}

func cloneDriverRows(rows [][]driver.Value) [][]driver.Value {
	cloned := make([][]driver.Value, len(rows))
	for index, row := range rows {
		cloned[index] = append([]driver.Value(nil), row...)
	}
	return cloned
}

func callNames(calls []sqlCall) []string {
	names := make([]string, len(calls))
	for index, call := range calls {
		names[index] = call.statement
	}
	return names
}

var _ data.Executor = (*sql.DB)(nil)
