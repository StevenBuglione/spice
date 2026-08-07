package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"slices"
	"sync"
	"testing"
)

var (
	errBegin    = errors.New("begin failed")
	errWork     = errors.New("work failed")
	errRollback = errors.New("rollback failed")
	errCommit   = errors.New("commit failed")
	errExec     = errors.New("exec failed")
)

func TestManagerCommitsSuccessfulWork(t *testing.T) {
	t.Parallel()
	state, db := openTestDatabase(t)
	observer := &recordingObserver{}
	manager, err := NewManager(db, observer)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	definition := Definition{
		ID:        "orders.PlaceOrder",
		Module:    "example.com/shop/orders",
		Isolation: sql.LevelSerializable,
		ReadOnly:  true,
	}

	err = manager.Within(context.Background(), definition, func(ctx context.Context, executor Executor) error {
		if value, ok := ctx.Value(observationContextKey{}).(string); !ok || value != definition.ID {
			return errors.New("observer context was not propagated")
		}
		contextExecutor, ok := ExecutorFromContext(ctx)
		if !ok || contextExecutor != executor {
			return errors.New("transaction executor was not propagated")
		}
		_, executeErr := executor.ExecContext(ctx, "INSERT INTO orders VALUES (?)", "order-1")
		return executeErr
	})
	if err != nil {
		t.Fatalf("Within() error = %v", err)
	}

	if got, want := state.operations(), []string{"begin", "exec", "commit"}; !slices.Equal(got, want) {
		t.Fatalf("operations = %v, want %v", got, want)
	}
	if state.options.Isolation != driver.IsolationLevel(sql.LevelSerializable) ||
		!state.options.ReadOnly {
		t.Fatalf("transaction options = %#v", state.options)
	}
	result := observer.onlyResult(t)
	if result.Definition != definition || result.Err != nil || result.Panicked || result.Duration < 0 {
		t.Fatalf("observation result = %#v", result)
	}
}

func TestExecutorFromContextRejectsMissingContext(t *testing.T) {
	t.Parallel()
	if executor, ok := ExecutorFromContext(nil); ok || executor != nil { //nolint:staticcheck // Nil context is an intentional fail-closed boundary case.
		t.Fatalf("ExecutorFromContext(nil) = %#v, %t", executor, ok)
	}
	if executor, ok := ExecutorFromContext(context.Background()); ok || executor != nil {
		t.Fatalf("ExecutorFromContext(background) = %#v, %t", executor, ok)
	}
}

func TestManagerRollsBackAndJoinsFailures(t *testing.T) {
	t.Parallel()
	state, db := openTestDatabase(t)
	state.rollbackErr = errRollback
	observer := &recordingObserver{}
	manager, err := NewManager(db, observer)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	err = manager.Within(context.Background(), validDefinition(), func(context.Context, Executor) error {
		return errWork
	})
	if !errors.Is(err, errWork) || !errors.Is(err, errRollback) {
		t.Fatalf("Within() error = %v, want joined work and rollback errors", err)
	}
	if got, want := state.operations(), []string{"begin", "rollback"}; !slices.Equal(got, want) {
		t.Fatalf("operations = %v, want %v", got, want)
	}
	if observed := observer.onlyResult(t); !errors.Is(observed.Err, errWork) ||
		!errors.Is(observed.Err, errRollback) {
		t.Fatalf("observed error = %v", observed.Err)
	}
}

func TestManagerRollsBackBeforeRepanic(t *testing.T) {
	t.Parallel()
	state, db := openTestDatabase(t)
	observer := &recordingObserver{}
	manager, err := NewManager(db, observer)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	panicValue := &struct{ message string }{message: "boom"}

	recovered := recoverWithin(func() {
		withinErr := manager.Within(
			context.Background(),
			validDefinition(),
			func(context.Context, Executor) error {
				panic(panicValue)
			},
		)
		t.Fatalf("Within() error = %v, want panic", withinErr)
	})
	if recovered != panicValue {
		t.Fatalf("recovered = %#v, want original panic value", recovered)
	}
	if got, want := state.operations(), []string{"begin", "rollback"}; !slices.Equal(got, want) {
		t.Fatalf("operations = %v, want %v", got, want)
	}
	result := observer.onlyResult(t)
	if !result.Panicked || !errors.Is(result.Err, ErrPanicked) {
		t.Fatalf("observation result = %#v", result)
	}
}

func TestManagerReportsDatabaseFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		configure  func(*databaseState)
		work       Work
		want       error
		operations []string
	}{
		{
			name: "begin",
			configure: func(state *databaseState) {
				state.beginErr = errBegin
			},
			work:       func(context.Context, Executor) error { return nil },
			want:       errBegin,
			operations: []string{"begin"},
		},
		{
			name: "execute",
			configure: func(state *databaseState) {
				state.execErr = errExec
			},
			work: func(ctx context.Context, executor Executor) error {
				_, err := executor.ExecContext(ctx, "UPDATE orders SET state = 'done'")
				return err
			},
			want:       errExec,
			operations: []string{"begin", "exec", "rollback"},
		},
		{
			name: "commit",
			configure: func(state *databaseState) {
				state.commitErr = errCommit
			},
			work:       func(context.Context, Executor) error { return nil },
			want:       errCommit,
			operations: []string{"begin", "commit"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			state, db := openTestDatabase(t)
			test.configure(state)
			observer := &recordingObserver{}
			manager, err := NewManager(db, observer)
			if err != nil {
				t.Fatalf("NewManager() error = %v", err)
			}
			err = manager.Within(context.Background(), validDefinition(), test.work)
			if !errors.Is(err, test.want) {
				t.Fatalf("Within() error = %v, want errors.Is(%v)", err, test.want)
			}
			if got := state.operations(); !slices.Equal(got, test.operations) {
				t.Fatalf("operations = %v, want %v", got, test.operations)
			}
			if observed := observer.onlyResult(t); !errors.Is(observed.Err, test.want) {
				t.Fatalf("observed error = %v, want errors.Is(%v)", observed.Err, test.want)
			}
		})
	}
}

func TestManagerHonorsCanceledContext(t *testing.T) {
	t.Parallel()
	state, db := openTestDatabase(t)
	observer := &recordingObserver{}
	manager, err := NewManager(db, observer)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false

	err = manager.Within(ctx, validDefinition(), func(context.Context, Executor) error {
		called = true
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Within() error = %v, want context.Canceled", err)
	}
	if called {
		t.Fatal("Within() called work for a canceled context")
	}
	if operations := state.operations(); len(operations) != 0 {
		t.Fatalf("operations = %v, want none", operations)
	}
	if observed := observer.onlyResult(t); !errors.Is(observed.Err, context.Canceled) {
		t.Fatalf("observed error = %v, want context.Canceled", observed.Err)
	}
}

func TestManagerValidatesInputs(t *testing.T) {
	t.Parallel()
	if _, err := NewManager(nil); err == nil {
		t.Fatal("NewManager(nil) error = nil")
	}
	_, db := openTestDatabase(t)
	var nilObserver *recordingObserver
	if _, err := NewManager(db, nilObserver); err == nil {
		t.Fatal("NewManager(typed nil observer) error = nil")
	}
	manager, err := NewManager(db)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if err := (*Manager)(nil).Within(context.Background(), validDefinition(), noWork); err == nil {
		t.Fatal("nil manager Within() error = nil")
	}
	if err := manager.Within(nilTestContext(), validDefinition(), noWork); err == nil {
		t.Fatal("Within(nil context) error = nil")
	}
	tests := []struct {
		name       string
		definition Definition
		work       Work
	}{
		{"missing ID", Definition{Module: "example.com/shop"}, noWork},
		{"missing module", Definition{ID: "operation"}, noWork},
		{
			"invalid isolation",
			Definition{ID: "operation", Module: "example.com/shop", Isolation: 99},
			noWork,
		},
		{"nil work", validDefinition(), nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := manager.Within(context.Background(), test.definition, test.work); err == nil {
				t.Fatal("Within() error = nil")
			}
		})
	}
}

func TestObserversNestAndIgnoreOptionalReturns(t *testing.T) {
	t.Parallel()
	_, db := openTestDatabase(t)
	var orderMu sync.Mutex
	var order []string
	first := observerFunc(func(ctx context.Context, _ Definition) (context.Context, func(Result)) {
		orderMu.Lock()
		order = append(order, "begin-first")
		orderMu.Unlock()
		return context.WithValue(ctx, observationContextKey{}, "first"), func(Result) {
			orderMu.Lock()
			order = append(order, "end-first")
			orderMu.Unlock()
		}
	})
	second := observerFunc(func(ctx context.Context, _ Definition) (context.Context, func(Result)) {
		if ctx.Value(observationContextKey{}) != "first" {
			t.Error("second observer did not receive first observer context")
		}
		orderMu.Lock()
		order = append(order, "begin-second")
		orderMu.Unlock()
		return nil, func(Result) {
			orderMu.Lock()
			order = append(order, "end-second")
			orderMu.Unlock()
		}
	})
	third := observerFunc(func(ctx context.Context, _ Definition) (context.Context, func(Result)) {
		if ctx.Value(observationContextKey{}) != "first" {
			t.Error("third observer did not receive retained observer context")
		}
		orderMu.Lock()
		order = append(order, "begin-third")
		orderMu.Unlock()
		return nil, nil
	})
	manager, err := NewManager(db, first, second, third)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if err := manager.Within(context.Background(), validDefinition(), noWork); err != nil {
		t.Fatalf("Within() error = %v", err)
	}
	if want := []string{
		"begin-first",
		"begin-second",
		"begin-third",
		"end-second",
		"end-first",
	}; !slices.Equal(order, want) {
		t.Fatalf("observer order = %v, want %v", order, want)
	}
}

func validDefinition() Definition {
	return Definition{ID: "orders.PlaceOrder", Module: "example.com/shop/orders"}
}

func noWork(context.Context, Executor) error {
	return nil
}

func nilTestContext() context.Context {
	return nil
}

func recoverWithin(run func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	run()
	return nil
}

type observationContextKey struct{}

type observerFunc func(context.Context, Definition) (context.Context, func(Result))

func (observe observerFunc) BeginTransaction(
	ctx context.Context,
	definition Definition,
) (context.Context, func(Result)) {
	return observe(ctx, definition)
}

type recordingObserver struct {
	mu      sync.Mutex
	results []Result
}

func (observer *recordingObserver) BeginTransaction(
	ctx context.Context,
	definition Definition,
) (context.Context, func(Result)) {
	return context.WithValue(ctx, observationContextKey{}, definition.ID), func(result Result) {
		observer.mu.Lock()
		observer.results = append(observer.results, result)
		observer.mu.Unlock()
	}
}

func (observer *recordingObserver) onlyResult(t *testing.T) Result {
	t.Helper()
	observer.mu.Lock()
	defer observer.mu.Unlock()
	if len(observer.results) != 1 {
		t.Fatalf("observation results = %#v, want one", observer.results)
	}
	return observer.results[0]
}

type databaseState struct {
	mu          sync.Mutex
	log         []string
	options     driver.TxOptions
	beginErr    error
	commitErr   error
	rollbackErr error
	execErr     error
}

func (state *databaseState) record(operation string) {
	state.mu.Lock()
	state.log = append(state.log, operation)
	state.mu.Unlock()
}

func (state *databaseState) operations() []string {
	state.mu.Lock()
	defer state.mu.Unlock()
	return append([]string(nil), state.log...)
}

type testConnector struct {
	state *databaseState
}

func (connector testConnector) Connect(context.Context) (driver.Conn, error) {
	return &testConnection{state: connector.state}, nil
}

func (connector testConnector) Driver() driver.Driver {
	return testDriver{}
}

type testDriver struct{}

func (testDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("test driver requires its connector")
}

type testConnection struct {
	state *databaseState
}

func (connection *testConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is unsupported by the test driver")
}

func (connection *testConnection) Close() error {
	return nil
}

func (connection *testConnection) Begin() (driver.Tx, error) {
	return connection.BeginTx(context.Background(), driver.TxOptions{})
}

func (connection *testConnection) BeginTx(
	_ context.Context,
	options driver.TxOptions,
) (driver.Tx, error) {
	connection.state.mu.Lock()
	connection.state.options = options
	err := connection.state.beginErr
	connection.state.log = append(connection.state.log, "begin")
	connection.state.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &testTransaction{state: connection.state}, nil
}

func (connection *testConnection) ExecContext(
	_ context.Context,
	_ string,
	_ []driver.NamedValue,
) (driver.Result, error) {
	connection.state.record("exec")
	if connection.state.execErr != nil {
		return nil, connection.state.execErr
	}
	return driver.RowsAffected(1), nil
}

type testTransaction struct {
	state *databaseState
}

func (tx *testTransaction) Commit() error {
	tx.state.record("commit")
	return tx.state.commitErr
}

func (tx *testTransaction) Rollback() error {
	tx.state.record("rollback")
	return tx.state.rollbackErr
}

func openTestDatabase(t *testing.T) (*databaseState, *sql.DB) {
	t.Helper()
	state := &databaseState{}
	db := sql.OpenDB(testConnector{state: state})
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("DB.Close() error = %v", err)
		}
	})
	return state, db
}

var (
	_ driver.Connector     = testConnector{}
	_ driver.Conn          = (*testConnection)(nil)
	_ driver.ConnBeginTx   = (*testConnection)(nil)
	_ driver.ExecerContext = (*testConnection)(nil)
	_ driver.Tx            = (*testTransaction)(nil)
	_ io.Closer            = (*testConnection)(nil)
)
