package migration

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

var errTest = errors.New("test failure")

func TestPlanNormalizesChecksumsAndSortsGlobalVersions(t *testing.T) {
	t.Parallel()
	specs := []Spec{
		{
			Version: 202607250002,
			Module:  "example.com/shop/inventory",
			Name:    "create inventory",
			SQL:     "CREATE TABLE inventory (id text);\r\n",
		},
		{
			Version: 202607250001,
			Module:  "example.com/shop/orders",
			Name:    "create orders",
			SQL:     "CREATE TABLE orders (id text);\n",
		},
	}
	plan, err := NewPlan(specs)
	if err != nil {
		t.Fatalf("NewPlan() error = %v", err)
	}
	migrations := plan.Migrations()
	if len(migrations) != 2 ||
		migrations[0].Version() != 202607250001 ||
		migrations[0].Module() != "example.com/shop/orders" ||
		migrations[0].Name() != "create orders" ||
		migrations[0].SQL() != "CREATE TABLE orders (id text);\n" ||
		len(migrations[0].Checksum()) != 64 ||
		migrations[1].SQL() != "CREATE TABLE inventory (id text);\n" {
		t.Fatalf("migrations = %#v", migrations)
	}
	migrations[0] = Migration{}
	if plan.Migrations()[0].Version() != 202607250001 {
		t.Fatal("Plan.Migrations() exposed plan storage")
	}
	normalized, err := NewPlan([]Spec{{
		Version: 1, Module: "module", Name: "name", SQL: "SELECT 1;\n",
	}})
	if err != nil {
		t.Fatalf("NewPlan(normalized) error = %v", err)
	}
	windows, err := NewPlan([]Spec{{
		Version: 1, Module: "module", Name: "name", SQL: "SELECT 1;\r\n",
	}})
	if err != nil {
		t.Fatalf("NewPlan(windows) error = %v", err)
	}
	if normalized.Migrations()[0].Checksum() != windows.Migrations()[0].Checksum() {
		t.Fatal("line-ending normalization changed checksum")
	}
}

func TestPlanRejectsInvalidAndDuplicateMigrations(t *testing.T) {
	t.Parallel()
	valid := Spec{Version: 1, Module: "module", Name: "name", SQL: "SELECT 1;"}
	tests := []Spec{
		{},
		{Version: 1, Name: "name", SQL: "SELECT 1;"},
		{Version: 1, Module: " module", Name: "name", SQL: "SELECT 1;"},
		{Version: 1, Module: "module", SQL: "SELECT 1;"},
		{Version: 1, Module: "module", Name: " name", SQL: "SELECT 1;"},
		{Version: 1, Module: "module", Name: "name"},
		{Version: 1, Module: "module", Name: "name", SQL: "SELECT\r1;"},
	}
	for index, spec := range tests {
		if _, err := NewPlan([]Spec{spec}); err == nil {
			t.Fatalf("NewPlan(case %d) error = nil", index)
		}
	}
	duplicate := valid
	duplicate.Module = "another"
	if _, err := NewPlan([]Spec{valid, duplicate}); err == nil {
		t.Fatal("NewPlan(duplicate version) error = nil")
	}
	if migrations := (*Plan)(nil).Migrations(); len(migrations) != 0 {
		t.Fatalf("nil Plan.Migrations() = %#v", migrations)
	}
}

func TestRunnerReconcilesPrefixAndAppliesPendingInOrder(t *testing.T) {
	t.Parallel()
	plan := testPlan(t)
	migrations := plan.Migrations()
	session := &fakeSession{applied: []Applied{appliedRecord(migrations[0])}}
	backend := backendFunc(func(
		ctx context.Context,
		work func(context.Context, Session) error,
	) error {
		return work(context.WithValue(ctx, testContextKey{}, "locked"), session)
	})
	var observations []Observation
	runner, err := NewRunner(backend, func(ctx context.Context, observation Observation) {
		if ctx.Value(testContextKey{}) != "locked" {
			t.Error("observer did not receive locked context")
		}
		observations = append(observations, observation)
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	result, err := runner.Run(context.Background(), plan)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result != (Result{Current: 1, Applied: 2}) ||
		!versionsEqual(session.executed, []uint64{2, 3}) {
		t.Fatalf("result = %#v, executed = %#v", result, session.executed)
	}
	if len(observations) != 2 ||
		observations[0].Version != 2 ||
		observations[0].Module != migrations[1].Module() ||
		observations[0].Name != migrations[1].Name() ||
		observations[0].Err != nil ||
		observations[0].Duration < 0 {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestRunnerRejectsRegistryDrift(t *testing.T) {
	t.Parallel()
	plan := testPlan(t)
	migrations := plan.Migrations()
	valid := appliedRecord(migrations[0])
	tests := [][]Applied{
		{valid, appliedRecord(migrations[1]), appliedRecord(migrations[2]), valid},
		{{Version: 2, Module: valid.Module, Name: valid.Name, Checksum: valid.Checksum, AppliedAt: valid.AppliedAt}},
		{{Version: valid.Version, Module: "another", Name: valid.Name, Checksum: valid.Checksum, AppliedAt: valid.AppliedAt}},
		{{Version: valid.Version, Module: valid.Module, Name: valid.Name, Checksum: "changed", AppliedAt: valid.AppliedAt}},
		{{Version: valid.Version, Module: valid.Module, Name: valid.Name, Checksum: valid.Checksum}},
	}
	for index, records := range tests {
		session := &fakeSession{applied: records}
		runner := newTestRunner(t, singleSessionBackend(session))
		if _, err := runner.Run(context.Background(), plan); err == nil {
			t.Fatalf("Run(case %d) error = nil", index)
		}
		if len(session.executed) != 0 {
			t.Fatalf("Run(case %d) executed migrations: %#v", index, session.executed)
		}
	}
}

func TestRunnerStopsAtFailureAndCancellation(t *testing.T) {
	t.Parallel()
	plan := testPlan(t)
	session := &fakeSession{applyErr: map[uint64]error{2: errTest}}
	var observations []Observation
	runner, err := NewRunner(singleSessionBackend(session), func(
		_ context.Context,
		observation Observation,
	) {
		observations = append(observations, observation)
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	result, err := runner.Run(context.Background(), plan)
	if !errors.Is(err, errTest) ||
		result != (Result{Applied: 1}) ||
		!versionsEqual(session.executed, []uint64{1, 2}) ||
		len(observations) != 2 ||
		!errors.Is(observations[1].Err, errTest) {
		t.Fatalf("Run() = %#v, %v, executed=%v, observations=%#v", result, err, session.executed, observations)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceledSession := &fakeSession{}
	canceledRunner := newTestRunner(t, singleSessionBackend(canceledSession))
	if _, err := canceledRunner.Run(ctx, plan); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run(canceled) error = %v", err)
	}
	if len(canceledSession.executed) != 0 {
		t.Fatalf("canceled execution = %#v", canceledSession.executed)
	}
}

func TestRunnerValidatesBackendProtocolAndFailures(t *testing.T) {
	t.Parallel()
	plan := testPlan(t)
	tests := []struct {
		name    string
		backend Backend
		want    error
	}{
		{
			name: "backend",
			backend: backendFunc(func(context.Context, func(context.Context, Session) error) error {
				return errTest
			}),
			want: errTest,
		},
		{
			name: "no invocation",
			backend: backendFunc(func(context.Context, func(context.Context, Session) error) error {
				return nil
			}),
		},
		{
			name: "multiple invocations",
			backend: backendFunc(func(ctx context.Context, work func(context.Context, Session) error) error {
				session := &fakeSession{}
				if err := work(ctx, session); err != nil {
					return err
				}
				return work(ctx, session)
			}),
		},
		{
			name: "nil context",
			backend: backendFunc(func(_ context.Context, work func(context.Context, Session) error) error {
				return work(nilContext(), &fakeSession{})
			}),
		},
		{
			name: "nil session",
			backend: backendFunc(func(ctx context.Context, work func(context.Context, Session) error) error {
				return work(ctx, nil)
			}),
		},
		{
			name:    "read registry",
			backend: singleSessionBackend(&fakeSession{readErr: errTest}),
			want:    errTest,
		},
	}
	for _, test := range tests {
		runner := newTestRunner(t, test.backend)
		_, err := runner.Run(context.Background(), plan)
		if err == nil || (test.want != nil && !errors.Is(err, test.want)) {
			t.Fatalf("%s: Run() error = %v", test.name, err)
		}
	}
}

func TestRunnerConstructionAndArguments(t *testing.T) {
	t.Parallel()
	if _, err := NewRunner(nil); err == nil {
		t.Fatal("NewRunner(nil) error = nil")
	}
	var typedNil *backendStub
	if _, err := NewRunner(typedNil); err == nil {
		t.Fatal("NewRunner(typed nil) error = nil")
	}
	if _, err := NewRunner(singleSessionBackend(&fakeSession{}), nil); err == nil {
		t.Fatal("NewRunner(nil observer) error = nil")
	}
	runner := newTestRunner(t, singleSessionBackend(&fakeSession{}))
	if _, err := runner.Run(nilContext(), testPlan(t)); err == nil {
		t.Fatal("Run(nil context) error = nil")
	}
	if _, err := runner.Run(context.Background(), nil); err == nil {
		t.Fatal("Run(nil plan) error = nil")
	}
	if _, err := (*Runner)(nil).Run(context.Background(), testPlan(t)); err == nil {
		t.Fatal("nil Run() error = nil")
	}
}

type testContextKey struct{}

type backendFunc func(context.Context, func(context.Context, Session) error) error

func (backend backendFunc) RunLocked(
	ctx context.Context,
	work func(context.Context, Session) error,
) error {
	return backend(ctx, work)
}

type backendStub struct{}

func (*backendStub) RunLocked(
	context.Context,
	func(context.Context, Session) error,
) error {
	return nil
}

type fakeSession struct {
	applied  []Applied
	readErr  error
	applyErr map[uint64]error
	executed []Migration
}

func (session *fakeSession) Applied(context.Context) ([]Applied, error) {
	return append([]Applied(nil), session.applied...), session.readErr
}

func (session *fakeSession) Apply(_ context.Context, migration Migration) error {
	session.executed = append(session.executed, migration)
	return session.applyErr[migration.Version()]
}

func singleSessionBackend(session Session) Backend {
	return backendFunc(func(
		ctx context.Context,
		work func(context.Context, Session) error,
	) error {
		return work(ctx, session)
	})
}

func newTestRunner(t *testing.T, backend Backend) *Runner {
	t.Helper()
	runner, err := NewRunner(backend)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	return runner
}

func testPlan(t *testing.T) *Plan {
	t.Helper()
	plan, err := NewPlan([]Spec{
		{Version: 3, Module: "payments", Name: "payments", SQL: "SELECT 3;"},
		{Version: 1, Module: "orders", Name: "orders", SQL: "SELECT 1;"},
		{Version: 2, Module: "inventory", Name: "inventory", SQL: "SELECT 2;"},
	})
	if err != nil {
		t.Fatalf("NewPlan() error = %v", err)
	}
	return plan
}

func appliedRecord(migration Migration) Applied {
	return Applied{
		Version:   migration.Version(),
		Module:    migration.Module(),
		Name:      migration.Name(),
		Checksum:  migration.Checksum(),
		AppliedAt: time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC),
	}
}

func versionsEqual(migrations []Migration, versions []uint64) bool {
	actual := make([]uint64, len(migrations))
	for index, migration := range migrations {
		actual[index] = migration.Version()
	}
	return slices.Equal(actual, versions)
}

func nilContext() context.Context {
	return nil
}

func TestErrorsExcludeSQLText(t *testing.T) {
	t.Parallel()
	secretSQL := "CREATE USER secret_password"
	plan, err := NewPlan([]Spec{{
		Version: 1, Module: "module", Name: "name", SQL: secretSQL,
	}})
	if err != nil {
		t.Fatalf("NewPlan() error = %v", err)
	}
	session := &fakeSession{applyErr: map[uint64]error{1: errTest}}
	runner := newTestRunner(t, singleSessionBackend(session))
	_, err = runner.Run(context.Background(), plan)
	if err == nil || strings.Contains(err.Error(), secretSQL) {
		t.Fatalf("Run() error = %v", err)
	}
}
