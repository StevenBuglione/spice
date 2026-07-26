package repository_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/StevenBuglione/spice/data"
	"github.com/StevenBuglione/spice/data/repository"
)

var (
	errQuery  = errors.New("query failed")
	errDecode = errors.New("decode failed")
	errRows   = errors.New("rows failed")
	errClose  = errors.New("close failed")
)

func TestQueryListPreservesOrderAndBoundsResults(t *testing.T) {
	t.Parallel()

	query := newStringQuery(t, 2, nil)
	database := openDatabase(t, queryResult{
		values: [][]driver.Value{{"first"}, {"second"}},
	})

	items, err := query.List(context.Background(), database)
	if err != nil {
		t.Fatalf("list rows: %v", err)
	}
	if strings.Join(items, ",") != "first,second" {
		t.Fatalf("unexpected items: %v", items)
	}

	database = openDatabase(t, queryResult{
		values: [][]driver.Value{{"first"}, {"second"}, {"third"}},
	})
	_, err = query.List(context.Background(), database)
	if !errors.Is(err, repository.ErrRowLimitExceeded) {
		t.Fatalf("expected row limit error, got %v", err)
	}
}

func TestQueryListReturnsAnOwnedEmptySlice(t *testing.T) {
	t.Parallel()

	query := newStringQuery(t, 2, nil)
	items, err := query.List(context.Background(), openDatabase(t, queryResult{}))
	if err != nil {
		t.Fatalf("list rows: %v", err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("expected owned empty slice, got %#v", items)
	}
}

func TestQueryOneAndOptionalEnforceCardinality(t *testing.T) {
	t.Parallel()

	query := newStringQuery(t, 5, nil)
	tests := []struct {
		name       string
		values     [][]driver.Value
		oneValue   string
		oneError   error
		optional   string
		found      bool
		optionalEr error
	}{
		{
			name:       "empty",
			oneError:   repository.ErrNotFound,
			optional:   "",
			found:      false,
			optionalEr: nil,
		},
		{
			name:     "one",
			values:   [][]driver.Value{{"value"}},
			oneValue: "value",
			optional: "value",
			found:    true,
		},
		{
			name:       "multiple",
			values:     [][]driver.Value{{"first"}, {"second"}},
			oneError:   repository.ErrMultipleRows,
			optionalEr: repository.ErrMultipleRows,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			value, err := query.One(
				context.Background(),
				openDatabase(t, queryResult{values: test.values}),
			)
			if value != test.oneValue || !errors.Is(err, test.oneError) {
				t.Fatalf("One() = (%q, %v), want (%q, %v)", value, err, test.oneValue, test.oneError)
			}

			value, found, err := query.Optional(
				context.Background(),
				openDatabase(t, queryResult{values: test.values}),
			)
			if value != test.optional || found != test.found || !errors.Is(err, test.optionalEr) {
				t.Fatalf(
					"Optional() = (%q, %t, %v), want (%q, %t, %v)",
					value,
					found,
					err,
					test.optional,
					test.found,
					test.optionalEr,
				)
			}
		})
	}
}

func TestQueryReportsDatabaseAndDecodeFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		result  queryResult
		decode  repository.Decoder[string]
		wantErr error
	}{
		{name: "query", result: queryResult{queryErr: errQuery}, wantErr: errQuery},
		{
			name:   "decode",
			result: queryResult{values: [][]driver.Value{{"value"}}},
			decode: func(repository.Scanner) (string, error) {
				return "", errDecode
			},
			wantErr: errDecode,
		},
		{
			name:    "iteration",
			result:  queryResult{values: [][]driver.Value{{"value"}}, rowsErr: errRows},
			wantErr: errRows,
		},
		{
			name:    "close",
			result:  queryResult{values: [][]driver.Value{{"value"}}, closeErr: errClose},
			wantErr: errClose,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			query := newStringQuery(t, 4, test.decode)
			_, err := query.List(context.Background(), openDatabase(t, test.result))
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestQueryFailureDoesNotExposeStatementOrArguments(t *testing.T) {
	t.Parallel()

	query, err := repository.NewQuery(repository.QuerySpec[string]{
		ID:        "accounts.find",
		Module:    "example.com/shop/accounts",
		Statement: "SELECT secret_column FROM secret_table",
		MaxRows:   1,
		Decode:    decodeString,
	})
	if err != nil {
		t.Fatalf("construct query: %v", err)
	}
	_, err = query.List(
		context.Background(),
		openDatabase(t, queryResult{queryErr: errQuery}),
		"secret-argument",
	)
	if err == nil {
		t.Fatal("expected query failure")
	}
	message := err.Error()
	for _, secret := range []string{"secret_column", "secret_table", "secret-argument"} {
		if strings.Contains(message, secret) {
			t.Fatalf("error exposed %q: %v", secret, err)
		}
	}
	if !strings.Contains(message, "accounts.find") {
		t.Fatalf("error omitted operation identity: %v", err)
	}
}

func TestQueryHonorsCancellationAndValidatesExecution(t *testing.T) {
	t.Parallel()

	query := newStringQuery(t, 2, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := query.List(ctx, openDatabase(t, queryResult{}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}

	var nilDatabase *sql.DB
	_, err = query.List(context.Background(), nilDatabase)
	if err == nil || !strings.Contains(err.Error(), "executor is nil") {
		t.Fatalf("expected typed nil executor error, got %v", err)
	}
	_, err = query.List(nilTestContext(), openDatabase(t, queryResult{}))
	if err == nil || !strings.Contains(err.Error(), "context is nil") {
		t.Fatalf("expected nil context error, got %v", err)
	}
	var nilQuery *repository.Query[string]
	_, err = nilQuery.List(context.Background(), openDatabase(t, queryResult{}))
	if err == nil || !strings.Contains(err.Error(), "query is nil") {
		t.Fatalf("expected nil query error, got %v", err)
	}
}

func TestNewQueryValidatesAndFreezesMetadata(t *testing.T) {
	t.Parallel()

	valid := repository.QuerySpec[string]{
		ID:        "orders.list",
		Module:    "example.com/shop/orders",
		Statement: "SELECT id FROM orders",
		MaxRows:   repository.DefaultMaxRows,
		Decode:    decodeString,
	}
	query, err := repository.NewQuery(valid)
	if err != nil {
		t.Fatalf("construct query: %v", err)
	}
	if query.ID() != valid.ID || query.Module() != valid.Module {
		t.Fatalf("unexpected metadata: ID=%q Module=%q", query.ID(), query.Module())
	}
	valid.ID = "changed"
	valid.Module = "changed"
	if query.ID() != "orders.list" || query.Module() != "example.com/shop/orders" {
		t.Fatal("query metadata changed with source specification")
	}

	tests := []struct {
		name   string
		mutate func(*repository.QuerySpec[string])
	}{
		{name: "ID", mutate: func(spec *repository.QuerySpec[string]) { spec.ID = "" }},
		{name: "module", mutate: func(spec *repository.QuerySpec[string]) { spec.Module = "" }},
		{name: "statement", mutate: func(spec *repository.QuerySpec[string]) { spec.Statement = "" }},
		{name: "zero max", mutate: func(spec *repository.QuerySpec[string]) { spec.MaxRows = 0 }},
		{name: "negative max", mutate: func(spec *repository.QuerySpec[string]) { spec.MaxRows = -1 }},
		{
			name: "excess max",
			mutate: func(spec *repository.QuerySpec[string]) {
				spec.MaxRows = repository.MaxRows + 1
			},
		},
		{name: "decoder", mutate: func(spec *repository.QuerySpec[string]) { spec.Decode = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			spec := valid
			test.mutate(&spec)
			if _, err := repository.NewQuery(spec); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	var nilQuery *repository.Query[string]
	if nilQuery.ID() != "" || nilQuery.Module() != "" {
		t.Fatal("nil query returned metadata")
	}
}

func newStringQuery(
	t *testing.T,
	maxRows int,
	decode repository.Decoder[string],
) *repository.Query[string] {
	t.Helper()
	if decode == nil {
		decode = decodeString
	}
	query, err := repository.NewQuery(repository.QuerySpec[string]{
		ID:        "orders.list",
		Module:    "example.com/shop/orders",
		Statement: "SELECT value FROM records",
		MaxRows:   maxRows,
		Decode:    decode,
	})
	if err != nil {
		t.Fatalf("construct query: %v", err)
	}
	return query
}

func decodeString(scanner repository.Scanner) (string, error) {
	var value string
	if err := scanner.Scan(&value); err != nil {
		return "", err
	}
	return value, nil
}

func nilTestContext() context.Context {
	return nil
}

type queryResult struct {
	values   [][]driver.Value
	queryErr error
	rowsErr  error
	closeErr error
}

type queryConnector struct {
	result queryResult
}

func (connector queryConnector) Connect(context.Context) (driver.Conn, error) {
	return &queryConnection{result: connector.result}, nil
}

func (connector queryConnector) Driver() driver.Driver {
	return queryDriver{}
}

type queryDriver struct{}

func (queryDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("query test driver requires a connector")
}

type queryConnection struct {
	result queryResult
}

func (connection *queryConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (connection *queryConnection) Close() error {
	return nil
}

func (connection *queryConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (connection *queryConnection) QueryContext(
	ctx context.Context,
	_ string,
	_ []driver.NamedValue,
) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if connection.result.queryErr != nil {
		return nil, connection.result.queryErr
	}
	return &queryRows{
		values:   connection.result.values,
		rowsErr:  connection.result.rowsErr,
		closeErr: connection.result.closeErr,
	}, nil
}

type queryRows struct {
	values   [][]driver.Value
	index    int
	rowsErr  error
	closeErr error
}

func (*queryRows) Columns() []string {
	return []string{"value"}
}

func (rows *queryRows) Close() error {
	return rows.closeErr
}

func (rows *queryRows) Next(destination []driver.Value) error {
	if rows.index < len(rows.values) {
		copy(destination, rows.values[rows.index])
		rows.index++
		return nil
	}
	if rows.rowsErr != nil {
		err := rows.rowsErr
		rows.rowsErr = nil
		return err
	}
	return io.EOF
}

func openDatabase(t *testing.T, result queryResult) *sql.DB {
	t.Helper()
	database := sql.OpenDB(queryConnector{result: result})
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return database
}

var _ data.Executor = (*sql.DB)(nil)
