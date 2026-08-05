package spicetest

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

	"github.com/spice-framework/spice/data"
)

var (
	errSQLBegin    = errors.New("begin failed")
	errSQLFactory  = errors.New("factory failed")
	errSQLRollback = errors.New("rollback failed")
)

func TestSQLSliceConstructsSubjectAndAlwaysRollsBack(t *testing.T) {
	t.Parallel()

	state, database := openSQLSliceDatabase(t)
	slice, err := NewSQL(
		context.Background(),
		database,
		func(ctx context.Context, executor data.Executor) (string, error) {
			if context.Cause(ctx) != nil {
				t.Fatal("factory context is canceled")
			}
			if _, execErr := executor.ExecContext(
				ctx,
				"INSERT",
				"argument",
			); execErr != nil {
				return "", execErr
			}
			return "subject", nil
		},
		SQLOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  true,
		},
	)
	if err != nil {
		t.Fatalf("NewSQL() error = %v", err)
	}
	if slice.Value() != "subject" ||
		slice.Executor() == nil ||
		slice.Closed() {
		t.Fatalf(
			"slice value=%q executor=%v closed=%t",
			slice.Value(),
			slice.Executor(),
			slice.Closed(),
		)
	}
	if state.options.Isolation != driver.IsolationLevel(sql.LevelSerializable) ||
		!state.options.ReadOnly {
		t.Fatalf("transaction options = %#v", state.options)
	}

	const callers = 16
	start := make(chan struct{})
	results := make(chan error, callers)
	for range callers {
		go func() {
			<-start
			results <- slice.Close()
		}()
	}
	close(start)
	for range callers {
		if closeErr := <-results; closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	}
	if !slice.Closed() ||
		!slices.Equal(state.operations(), []string{"begin", "exec", "rollback"}) {
		t.Fatalf(
			"closed=%t operations=%v",
			slice.Closed(),
			state.operations(),
		)
	}
}

func TestSQLSliceRollsBackFactoryFailureAndPanic(t *testing.T) {
	t.Parallel()

	state, database := openSQLSliceDatabase(t)
	state.rollbackErr = errSQLRollback
	if _, err := NewSQL(
		context.Background(),
		database,
		func(context.Context, data.Executor) (string, error) {
			return "", errSQLFactory
		},
		SQLOptions{},
	); !errors.Is(err, errSQLFactory) || !errors.Is(err, errSQLRollback) {
		t.Fatalf("NewSQL(factory failure) error = %v", err)
	}
	if !slices.Equal(state.operations(), []string{"begin", "rollback"}) {
		t.Fatalf("factory failure operations = %v", state.operations())
	}

	panicState, panicDatabase := openSQLSliceDatabase(t)
	panicValue := &struct{ ID string }{ID: "original"}
	recovered := func() (value any) {
		defer func() {
			value = recover()
		}()
		if _, constructionErr := NewSQL(
			context.Background(),
			panicDatabase,
			func(context.Context, data.Executor) (string, error) {
				panic(panicValue)
			},
			SQLOptions{},
		); constructionErr != nil {
			t.Fatalf("NewSQL(panic) error = %v", constructionErr)
		}
		return nil
	}()
	if recovered != panicValue {
		t.Fatalf("recovered panic = %#v, want original", recovered)
	}
	if !slices.Equal(panicState.operations(), []string{"begin", "rollback"}) {
		t.Fatalf("panic operations = %v", panicState.operations())
	}

	doubleState, doubleDatabase := openSQLSliceDatabase(t)
	doubleState.rollbackErr = errSQLRollback
	doubleFailure := func() (value any) {
		defer func() {
			value = recover()
		}()
		if _, constructionErr := NewSQL(
			context.Background(),
			doubleDatabase,
			func(context.Context, data.Executor) (string, error) {
				panic(panicValue)
			},
			SQLOptions{},
		); constructionErr != nil {
			t.Fatalf("NewSQL(double failure) error = %v", constructionErr)
		}
		return nil
	}()
	typed, typedOK := doubleFailure.(*SQLRollbackPanic)
	if !typedOK ||
		typed.Value != panicValue ||
		!errors.Is(typed, errSQLRollback) ||
		strings.Contains(typed.Error(), "original") {
		t.Fatalf("double failure = %#v", doubleFailure)
	}
}

func TestSQLSliceRejectsInvalidInputsAndBeginFailure(t *testing.T) {
	t.Parallel()

	factory := func(context.Context, data.Executor) (string, error) {
		return "subject", nil
	}
	_, database := openSQLSliceDatabase(t)
	if _, err := NewSQL(
		nilTestContext(),
		database,
		factory,
		SQLOptions{},
	); err == nil {
		t.Fatal("NewSQL(nil context) unexpectedly succeeded")
	}
	if _, err := NewSQL(
		context.Background(),
		nil,
		factory,
		SQLOptions{},
	); err == nil {
		t.Fatal("NewSQL(nil database) unexpectedly succeeded")
	}
	if _, err := NewSQL[string](
		context.Background(),
		database,
		nil,
		SQLOptions{},
	); err == nil {
		t.Fatal("NewSQL(nil factory) unexpectedly succeeded")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewSQL(
		canceled,
		database,
		factory,
		SQLOptions{},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("NewSQL(canceled) error = %v", err)
	}
	if _, err := NewSQL(
		context.Background(),
		database,
		factory,
		SQLOptions{Isolation: sql.IsolationLevel(99)},
	); err == nil {
		t.Fatal("NewSQL(invalid isolation) unexpectedly succeeded")
	}

	beginState, beginDatabase := openSQLSliceDatabase(t)
	beginState.beginErr = errSQLBegin
	if _, err := NewSQL(
		context.Background(),
		beginDatabase,
		factory,
		SQLOptions{},
	); !errors.Is(err, errSQLBegin) {
		t.Fatalf("NewSQL(begin failure) error = %v", err)
	}
	if !slices.Equal(beginState.operations(), []string{"begin"}) {
		t.Fatalf("begin failure operations = %v", beginState.operations())
	}
}

func TestSQLSliceReportsRollbackFailureAndNilAccessors(t *testing.T) {
	t.Parallel()

	state, database := openSQLSliceDatabase(t)
	state.rollbackErr = errSQLRollback
	slice, err := NewSQL(
		context.Background(),
		database,
		func(context.Context, data.Executor) (int, error) {
			return 41, nil
		},
		SQLOptions{},
	)
	if err != nil {
		t.Fatalf("NewSQL() error = %v", err)
	}
	if err := slice.Close(); !errors.Is(err, errSQLRollback) {
		t.Fatalf("Close() error = %v, want rollback failure", err)
	}
	if err := slice.Close(); !errors.Is(err, errSQLRollback) {
		t.Fatalf("second Close() error = %v, want rollback failure", err)
	}

	var nilSlice *SQL[int]
	if nilSlice.Value() != 0 ||
		nilSlice.Executor() != nil ||
		!nilSlice.Closed() ||
		nilSlice.Close() != nil {
		t.Fatal("nil SQL slice accessors returned unexpected values")
	}
}

type sqlSliceState struct {
	mu          sync.Mutex
	log         []string
	options     driver.TxOptions
	beginErr    error
	rollbackErr error
}

func (state *sqlSliceState) record(operation string) {
	state.mu.Lock()
	state.log = append(state.log, operation)
	state.mu.Unlock()
}

func (state *sqlSliceState) operations() []string {
	state.mu.Lock()
	defer state.mu.Unlock()
	return slices.Clone(state.log)
}

type sqlSliceConnector struct{ state *sqlSliceState }

func (connector sqlSliceConnector) Connect(context.Context) (driver.Conn, error) {
	return &sqlSliceConnection{state: connector.state}, nil
}

func (connector sqlSliceConnector) Driver() driver.Driver {
	return sqlSliceDriver{}
}

type sqlSliceDriver struct{}

func (sqlSliceDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use SQL slice connector")
}

type sqlSliceConnection struct{ state *sqlSliceState }

func (*sqlSliceConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}

func (*sqlSliceConnection) Close() error { return nil }

func (connection *sqlSliceConnection) Begin() (driver.Tx, error) {
	return connection.BeginTx(context.Background(), driver.TxOptions{})
}

func (connection *sqlSliceConnection) BeginTx(
	_ context.Context,
	options driver.TxOptions,
) (driver.Tx, error) {
	connection.state.mu.Lock()
	connection.state.options = options
	connection.state.log = append(connection.state.log, "begin")
	err := connection.state.beginErr
	connection.state.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &sqlSliceTransaction{state: connection.state}, nil
}

func (connection *sqlSliceConnection) ExecContext(
	_ context.Context,
	_ string,
	_ []driver.NamedValue,
) (driver.Result, error) {
	connection.state.record("exec")
	return driver.RowsAffected(1), nil
}

type sqlSliceTransaction struct{ state *sqlSliceState }

func (transaction *sqlSliceTransaction) Commit() error {
	transaction.state.record("commit")
	return nil
}

func (transaction *sqlSliceTransaction) Rollback() error {
	transaction.state.record("rollback")
	return transaction.state.rollbackErr
}

func openSQLSliceDatabase(t *testing.T) (*sqlSliceState, *sql.DB) {
	t.Helper()
	state := &sqlSliceState{}
	database := sql.OpenDB(sqlSliceConnector{state: state})
	database.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("database Close() error = %v", err)
		}
	})
	return state, database
}

var (
	_ driver.Connector     = sqlSliceConnector{}
	_ driver.Conn          = (*sqlSliceConnection)(nil)
	_ driver.ConnBeginTx   = (*sqlSliceConnection)(nil)
	_ driver.ExecerContext = (*sqlSliceConnection)(nil)
	_ driver.Tx            = (*sqlSliceTransaction)(nil)
	_ io.Closer            = (*sqlSliceConnection)(nil)
)
