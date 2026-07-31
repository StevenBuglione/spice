package security

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/StevenBuglione/spice/web"
)

func TestAuthorizerAllowsExactRoleAndScopePolicy(t *testing.T) {
	t.Parallel()
	policy := newTestPolicy(t, PolicySpec{
		Definition: Definition{ID: "orders.read", Module: "example.com/shop/orders"},
		AnyRoles:   []string{"support", "admin", "admin"},
		AllRoles:   []string{"user"},
		AllScopes:  []string{"orders:read"},
	})
	roles := []string{"user", "admin"}
	principal, err := NewPrincipal("subject-1", "https://issuer.example", roles, []string{"orders:read"})
	if err != nil {
		t.Fatalf("NewPrincipal() error = %v", err)
	}
	roles[0] = "mutated"
	returned := principal.Roles()
	returned[0] = "mutated"
	if principal.Subject() != "subject-1" ||
		principal.Issuer() != "https://issuer.example" ||
		!slices.Equal(principal.Roles(), []string{"admin", "user"}) ||
		!slices.Equal(principal.Scopes(), []string{"orders:read"}) {
		t.Fatalf("principal = %#v", principal)
	}
	ctx, err := WithPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatalf("WithPrincipal() error = %v", err)
	}
	authorizer, err := NewAuthorizer()
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	if err := authorizer.Authorize(ctx, policy); err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if policy.Definition() != (Definition{ID: "orders.read", Module: "example.com/shop/orders"}) {
		t.Fatalf("Policy.Definition() = %#v", policy.Definition())
	}
}

func TestAuthorizerEvaluatesRestrictedExpression(t *testing.T) {
	t.Parallel()
	policy, err := NewPolicy(PolicySpec{
		Definition: Definition{ID: "owner-or-admin", Module: "orders"},
		Expression: `authenticated && (subject == "owner" || hasRole("admin")) && hasScope("orders:read")`,
	})
	if err != nil {
		t.Fatalf("NewPolicy() error = %v", err)
	}
	if policy.Expression() == "" {
		t.Fatal("Expression() is empty")
	}
	authorizer, err := NewAuthorizer()
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	principal, err := NewPrincipal("owner", "issuer", nil, []string{"orders:read"})
	if err != nil {
		t.Fatalf("NewPrincipal() error = %v", err)
	}
	ctx, err := WithPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatalf("WithPrincipal() error = %v", err)
	}
	if authorizeErr := authorizer.Authorize(ctx, policy); authorizeErr != nil {
		t.Fatalf("Authorize(owner) error = %v", authorizeErr)
	}

	principal, err = NewPrincipal("other", "issuer", []string{"reader"}, []string{"orders:read"})
	if err != nil {
		t.Fatalf("NewPrincipal(reader) error = %v", err)
	}
	ctx, err = WithPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatalf("WithPrincipal(reader) error = %v", err)
	}
	err = authorizer.Authorize(ctx, policy)
	denied, ok := errors.AsType[*DeniedError](err)
	if !ok || denied.Reason != ReasonExpression {
		t.Fatalf("Authorize(reader) error = %#v", err)
	}
}

func TestAuthorizationExpressionValidation(t *testing.T) {
	t.Parallel()
	if err := ValidateExpression(`hasRole("admin")`); err != nil {
		t.Fatalf("ValidateExpression() error = %v", err)
	}
	for _, source := range []string{
		`principal.subject == "owner"`,
		`bean("admin")`,
		`subject = "owner"`,
	} {
		if err := ValidateExpression(source); err == nil {
			t.Fatalf("ValidateExpression(%q) error = nil", source)
		}
	}
	if _, err := NewPolicy(PolicySpec{
		Definition: Definition{ID: "invalid", Module: "orders"},
		Expression: " authenticated",
	}); err == nil {
		t.Fatal("NewPolicy() accepted surrounding whitespace")
	}
}

func TestAuthorizerDeniesAnonymousRoleAndScopeFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		spec      PolicySpec
		principal *Principal
		reason    Reason
		status    int
	}{
		{
			name:   "anonymous",
			spec:   PolicySpec{Definition: testDefinition(), Authenticated: true},
			reason: ReasonUnauthenticated,
			status: http.StatusUnauthorized,
		},
		{
			name: "role",
			spec: PolicySpec{
				Definition: testDefinition(),
				AllRoles:   []string{"admin"},
			},
			principal: testPrincipal(t, []string{"user"}, nil),
			reason:    ReasonRole,
			status:    http.StatusForbidden,
		},
		{
			name: "scope",
			spec: PolicySpec{
				Definition: testDefinition(),
				AllScopes:  []string{"orders:write"},
			},
			principal: testPrincipal(t, nil, []string{"orders:read"}),
			reason:    ReasonScope,
			status:    http.StatusForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			policy := newTestPolicy(t, test.spec)
			ctx := context.Background()
			if test.principal != nil {
				var err error
				ctx, err = WithPrincipal(ctx, *test.principal)
				if err != nil {
					t.Fatalf("WithPrincipal() error = %v", err)
				}
			}
			authorizer, err := NewAuthorizer()
			if err != nil {
				t.Fatalf("NewAuthorizer() error = %v", err)
			}
			err = authorizer.Authorize(ctx, policy)
			denied, ok := errors.AsType[*DeniedError](err)
			if !ok || denied.Reason != test.reason || denied.Problem().Status != test.status {
				t.Fatalf("Authorize() error = %#v", err)
			}
			if strings.Contains(err.Error(), "subject-1") ||
				strings.Contains(err.Error(), "orders:") {
				t.Fatalf("Authorize() leaked claims: %v", err)
			}
		})
	}
}

func TestAuthorizerObservesBoundedDecisions(t *testing.T) {
	t.Parallel()
	var decisionsMu sync.Mutex
	var decisions []Decision
	authorizer, err := NewAuthorizer(func(_ context.Context, decision Decision) {
		decisionsMu.Lock()
		decisions = append(decisions, decision)
		decisionsMu.Unlock()
	})
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	policy := newTestPolicy(t, PolicySpec{Definition: testDefinition(), Authenticated: true})
	if authErr := authorizer.Authorize(context.Background(), policy); authErr == nil {
		t.Fatal("Authorize(anonymous) error = nil")
	}
	ctx, err := WithPrincipal(context.Background(), *testPrincipal(t, nil, nil))
	if err != nil {
		t.Fatalf("WithPrincipal() error = %v", err)
	}
	if authErr := authorizer.Authorize(ctx, policy); authErr != nil {
		t.Fatalf("Authorize(authenticated) error = %v", authErr)
	}
	decisionsMu.Lock()
	defer decisionsMu.Unlock()
	if len(decisions) != 2 ||
		decisions[0].Reason != ReasonUnauthenticated ||
		decisions[0].Allowed ||
		decisions[1].Reason != ReasonAllowed ||
		!decisions[1].Allowed ||
		decisions[1].Duration < 0 {
		t.Fatalf("decisions = %#v", decisions)
	}
}

func TestGuardWritesSafeProblemsAndCallsAuthorizedHandler(t *testing.T) {
	t.Parallel()
	authorizer, err := NewAuthorizer()
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	policy := newTestPolicy(t, PolicySpec{Definition: testDefinition(), Authenticated: true})
	guard, err := Guard(authorizer, policy, nil)
	if err != nil {
		t.Fatalf("Guard() error = %v", err)
	}
	handler, err := web.Chain(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}), guard)
	if err != nil {
		t.Fatalf("web.Chain() error = %v", err)
	}

	anonymous := httptest.NewRecorder()
	handler.ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "/orders", nil))
	if anonymous.Code != http.StatusUnauthorized ||
		anonymous.Header().Get("WWW-Authenticate") != "Bearer" ||
		anonymous.Header().Get("Content-Type") != "application/problem+json" ||
		strings.Contains(anonymous.Body.String(), testDefinition().ID) {
		t.Fatalf("anonymous response = %d %s", anonymous.Code, anonymous.Body.String())
	}

	ctx, err := WithPrincipal(context.Background(), *testPrincipal(t, nil, nil))
	if err != nil {
		t.Fatalf("WithPrincipal() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/orders", nil).WithContext(ctx)
	authorized := httptest.NewRecorder()
	handler.ServeHTTP(authorized, request)
	if authorized.Code != http.StatusNoContent {
		t.Fatalf("authorized status = %d", authorized.Code)
	}
}

func TestGuardReportsResponseWriteFailure(t *testing.T) {
	t.Parallel()
	authorizer, err := NewAuthorizer()
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	policy := newTestPolicy(t, PolicySpec{Definition: testDefinition(), Authenticated: true})
	var writeErr error
	guard, err := Guard(authorizer, policy, func(_ context.Context, err error) {
		writeErr = err
	})
	if err != nil {
		t.Fatalf("Guard() error = %v", err)
	}
	handler := guard(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("authorized handler called")
	}))
	handler.ServeHTTP(&failingWriter{header: make(http.Header)}, httptest.NewRequest(
		http.MethodGet,
		"/orders",
		nil,
	))
	if writeErr == nil {
		t.Fatal("write failure was not reported")
	}
}

func TestPolicyPrincipalAndAuthorizerValidation(t *testing.T) {
	t.Parallel()
	policies := []PolicySpec{
		{Definition: Definition{Module: "module"}, Authenticated: true},
		{Definition: Definition{ID: "policy"}, Authenticated: true},
		{Definition: Definition{ID: " policy", Module: "module"}, Authenticated: true},
		{Definition: Definition{ID: "policy", Module: "module "}, Authenticated: true},
		{Definition: testDefinition()},
		{Definition: testDefinition(), AnyRoles: []string{""}},
		{Definition: testDefinition(), AllRoles: []string{" admin"}},
		{Definition: testDefinition(), AllScopes: []string{"scope "}},
	}
	for index, spec := range policies {
		if _, err := NewPolicy(spec); err == nil {
			t.Fatalf("NewPolicy(case %d) error = nil", index)
		}
	}
	if _, err := NewPrincipal("", "issuer", nil, nil); err == nil {
		t.Fatal("NewPrincipal(missing subject) error = nil")
	}
	if _, err := NewPrincipal(" subject", "issuer", nil, nil); err == nil {
		t.Fatal("NewPrincipal(invalid subject) error = nil")
	}
	if _, err := NewPrincipal("subject", "", nil, nil); err == nil {
		t.Fatal("NewPrincipal(missing issuer) error = nil")
	}
	if _, err := NewPrincipal("subject", "issuer ", nil, nil); err == nil {
		t.Fatal("NewPrincipal(invalid issuer) error = nil")
	}
	if _, err := NewPrincipal("subject", "issuer", []string{""}, nil); err == nil {
		t.Fatal("NewPrincipal(invalid role) error = nil")
	}
	if _, err := NewPrincipal("subject", "issuer", nil, []string{" scope"}); err == nil {
		t.Fatal("NewPrincipal(invalid scope) error = nil")
	}
	if _, err := NewAuthorizer(nil); err == nil {
		t.Fatal("NewAuthorizer(nil observer) error = nil")
	}
	authorizer, err := NewAuthorizer()
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	if err := authorizer.Authorize(nilTestContext(), Policy{}); err == nil {
		t.Fatal("Authorize(nil context) error = nil")
	}
	if err := (*Authorizer)(nil).Authorize(context.Background(), Policy{}); err == nil {
		t.Fatal("nil Authorize() error = nil")
	}
	if err := authorizer.Authorize(context.Background(), Policy{}); err == nil {
		t.Fatal("Authorize(invalid policy) error = nil")
	}
	if _, err := WithPrincipal(nilTestContext(), Principal{}); err == nil {
		t.Fatal("WithPrincipal(nil context) error = nil")
	}
	if _, err := WithPrincipal(context.Background(), Principal{}); err == nil {
		t.Fatal("WithPrincipal(invalid principal) error = nil")
	}
	if _, ok := PrincipalFromContext(nilTestContext()); ok {
		t.Fatal("PrincipalFromContext(nil) found a principal")
	}
	if _, err := Guard(nil, Policy{}, nil); err == nil {
		t.Fatal("Guard(nil authorizer) error = nil")
	}
	if _, err := Guard(authorizer, Policy{}, nil); err == nil {
		t.Fatal("Guard(invalid policy) error = nil")
	}
	if (*DeniedError)(nil).Error() != "authorization denied" ||
		(*DeniedError)(nil).Problem().Status != http.StatusForbidden {
		t.Fatal("nil DeniedError contract changed")
	}
}

type failingWriter struct {
	header http.Header
}

func (writer *failingWriter) Header() http.Header {
	return writer.header
}

func (*failingWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func (*failingWriter) WriteHeader(int) {}

func newTestPolicy(t *testing.T, spec PolicySpec) Policy {
	t.Helper()
	policy, err := NewPolicy(spec)
	if err != nil {
		t.Fatalf("NewPolicy() error = %v", err)
	}
	return policy
}

func testDefinition() Definition {
	return Definition{ID: "orders.read", Module: "example.com/shop/orders"}
}

func testPrincipal(t *testing.T, roles, scopes []string) *Principal {
	t.Helper()
	principal, err := NewPrincipal("subject-1", "https://issuer.example", roles, scopes)
	if err != nil {
		t.Fatalf("NewPrincipal() error = %v", err)
	}
	return &principal
}

func nilTestContext() context.Context {
	return nil
}
