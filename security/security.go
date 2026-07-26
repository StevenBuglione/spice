// Package security provides immutable authenticated principals and
// deny-by-default authorization policies for generated Spice guards.
package security

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/StevenBuglione/spice/web"
)

// Definition identifies one compiler-owned authorization policy and module.
type Definition struct {
	ID     string
	Module string
}

// PolicySpec is the inspectable generated input to NewPolicy. Role and scope
// names are exact and case-sensitive.
type PolicySpec struct {
	Definition    Definition
	Authenticated bool
	AnyRoles      []string
	AllRoles      []string
	AllScopes     []string
}

// Policy is an immutable validated authorization policy.
type Policy struct {
	definition    Definition
	authenticated bool
	anyRoles      []string
	allRoles      []string
	allScopes     []string
}

// NewPolicy validates and freezes one generated policy. A policy with no
// requirement is rejected so zero configuration can never grant access.
func NewPolicy(spec PolicySpec) (Policy, error) {
	if spec.Definition.ID == "" || strings.TrimSpace(spec.Definition.ID) != spec.Definition.ID {
		return Policy{}, errors.New("construct authorization policy: policy ID is required")
	}
	if spec.Definition.Module == "" ||
		strings.TrimSpace(spec.Definition.Module) != spec.Definition.Module {
		return Policy{}, fmt.Errorf(
			"construct authorization policy %q: module is required",
			spec.Definition.ID,
		)
	}
	anyRoles, err := normalizeNames(spec.Definition.ID, "any role", spec.AnyRoles)
	if err != nil {
		return Policy{}, err
	}
	allRoles, err := normalizeNames(spec.Definition.ID, "required role", spec.AllRoles)
	if err != nil {
		return Policy{}, err
	}
	allScopes, err := normalizeNames(spec.Definition.ID, "required scope", spec.AllScopes)
	if err != nil {
		return Policy{}, err
	}
	if !spec.Authenticated &&
		len(anyRoles) == 0 &&
		len(allRoles) == 0 &&
		len(allScopes) == 0 {
		return Policy{}, fmt.Errorf(
			"construct authorization policy %q: at least one requirement is required",
			spec.Definition.ID,
		)
	}
	return Policy{
		definition:    spec.Definition,
		authenticated: spec.Authenticated,
		anyRoles:      anyRoles,
		allRoles:      allRoles,
		allScopes:     allScopes,
	}, nil
}

// Definition returns the policy's stable generated identity.
func (policy Policy) Definition() Definition {
	return policy.definition
}

// Principal is an immutable identity created only after authentication.
type Principal struct {
	subject string
	issuer  string
	roles   []string
	scopes  []string
}

// NewPrincipal validates and freezes verified identity claims.
func NewPrincipal(subject, issuer string, roles, scopes []string) (Principal, error) {
	if subject == "" || strings.TrimSpace(subject) != subject {
		return Principal{}, errors.New("construct principal: subject is required")
	}
	if issuer == "" || strings.TrimSpace(issuer) != issuer {
		return Principal{}, errors.New("construct principal: issuer is required")
	}
	normalizedRoles, err := normalizeClaims("role", roles)
	if err != nil {
		return Principal{}, err
	}
	normalizedScopes, err := normalizeClaims("scope", scopes)
	if err != nil {
		return Principal{}, err
	}
	return Principal{
		subject: subject,
		issuer:  issuer,
		roles:   normalizedRoles,
		scopes:  normalizedScopes,
	}, nil
}

// Subject returns the verified subject identifier.
func (principal Principal) Subject() string {
	return principal.subject
}

// Issuer returns the verified issuer identifier.
func (principal Principal) Issuer() string {
	return principal.issuer
}

// Roles returns a defensive copy of exact verified roles.
func (principal Principal) Roles() []string {
	return append([]string(nil), principal.roles...)
}

// Scopes returns a defensive copy of exact verified scopes.
func (principal Principal) Scopes() []string {
	return append([]string(nil), principal.scopes...)
}

type principalContextKey struct{}

// WithPrincipal returns a derived context containing a validated principal.
func WithPrincipal(ctx context.Context, principal Principal) (context.Context, error) {
	if ctx == nil {
		return nil, errors.New("attach principal: context is nil")
	}
	if principal.subject == "" || principal.issuer == "" {
		return nil, errors.New("attach principal: principal is invalid")
	}
	return context.WithValue(ctx, principalContextKey{}, principal), nil
}

// PrincipalFromContext returns the authenticated principal, if present.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok && principal.subject != "" && principal.issuer != ""
}

// Reason is a stable authorization decision class.
type Reason string

const (
	// ReasonAllowed identifies a satisfied policy.
	ReasonAllowed Reason = "allowed"
	// ReasonUnauthenticated identifies a missing principal.
	ReasonUnauthenticated Reason = "unauthenticated"
	// ReasonRole identifies an unmet role requirement.
	ReasonRole Reason = "role"
	// ReasonScope identifies an unmet scope requirement.
	ReasonScope Reason = "scope"
)

// Decision contains bounded policy metadata and no identity claims.
type Decision struct {
	Definition Definition
	Allowed    bool
	Reason     Reason
	Duration   time.Duration
}

// Observer receives completed authorization decisions synchronously.
type Observer func(context.Context, Decision)

// Authorizer evaluates immutable policies without a global security context.
type Authorizer struct {
	observers []Observer
}

// NewAuthorizer constructs an instance-owned evaluator.
func NewAuthorizer(observers ...Observer) (*Authorizer, error) {
	for index, observer := range observers {
		if observer == nil {
			return nil, fmt.Errorf("construct authorizer: observer %d is nil", index)
		}
	}
	return &Authorizer{observers: append([]Observer(nil), observers...)}, nil
}

// Authorize evaluates one policy against the principal in ctx.
func (authorizer *Authorizer) Authorize(ctx context.Context, policy Policy) error {
	if ctx == nil {
		return errors.New("authorize: context is nil")
	}
	if authorizer == nil {
		return errors.New("authorize: authorizer is nil")
	}
	if err := validateFrozenPolicy(policy); err != nil {
		return err
	}
	started := time.Now()
	principal, authenticated := PrincipalFromContext(ctx)
	reason := evaluate(policy, principal, authenticated)
	decision := Decision{
		Definition: policy.definition,
		Allowed:    reason == ReasonAllowed,
		Reason:     reason,
		Duration:   time.Since(started),
	}
	for _, observer := range authorizer.observers {
		observer(ctx, decision)
	}
	if decision.Allowed {
		return nil
	}
	return &DeniedError{Definition: policy.definition, Reason: reason}
}

// DeniedError is a safe authorization failure.
type DeniedError struct {
	Definition Definition
	Reason     Reason
}

// Error returns no principal or claim data.
func (err *DeniedError) Error() string {
	if err == nil {
		return "authorization denied"
	}
	if err.Reason == ReasonUnauthenticated {
		return fmt.Sprintf("authorization policy %q requires authentication", err.Definition.ID)
	}
	return fmt.Sprintf("authorization policy %q denied access", err.Definition.ID)
}

// Problem returns a safe RFC 9457 response.
func (err *DeniedError) Problem() web.Problem {
	if err != nil && err.Reason == ReasonUnauthenticated {
		return web.Problem{
			Type:   "https://spice.dev/problems/unauthenticated",
			Title:  "Authentication required",
			Status: http.StatusUnauthorized,
		}
	}
	return web.Problem{
		Type:   "https://spice.dev/problems/forbidden",
		Title:  "Forbidden",
		Status: http.StatusForbidden,
	}
}

// WriteFailure receives an HTTP response-write failure that cannot be returned
// through http.Handler.
type WriteFailure func(context.Context, error)

// Guard constructs HTTP authorization middleware after validating its policy.
func Guard(
	authorizer *Authorizer,
	policy Policy,
	onWriteFailure WriteFailure,
) (web.Middleware, error) {
	if authorizer == nil {
		return nil, errors.New("construct authorization guard: authorizer is nil")
	}
	if err := validateFrozenPolicy(policy); err != nil {
		return nil, err
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if err := authorizer.Authorize(request.Context(), policy); err != nil {
				if denied, ok := errors.AsType[*DeniedError](err); ok &&
					denied.Reason == ReasonUnauthenticated {
					writer.Header().Set("WWW-Authenticate", "Bearer")
				}
				if writeErr := web.WriteError(writer, request, err, nil); writeErr != nil &&
					onWriteFailure != nil {
					onWriteFailure(request.Context(), writeErr)
				}
				return
			}
			next.ServeHTTP(writer, request)
		})
	}, nil
}

func evaluate(policy Policy, principal Principal, authenticated bool) Reason {
	requiresPrincipal := policy.authenticated ||
		len(policy.anyRoles) != 0 ||
		len(policy.allRoles) != 0 ||
		len(policy.allScopes) != 0
	if requiresPrincipal && !authenticated {
		return ReasonUnauthenticated
	}
	if len(policy.anyRoles) != 0 && !containsAny(principal.roles, policy.anyRoles) {
		return ReasonRole
	}
	if !containsAll(principal.roles, policy.allRoles) {
		return ReasonRole
	}
	if !containsAll(principal.scopes, policy.allScopes) {
		return ReasonScope
	}
	return ReasonAllowed
}

func containsAny(actual, required []string) bool {
	for _, item := range required {
		if _, found := slices.BinarySearch(actual, item); found {
			return true
		}
	}
	return false
}

func containsAll(actual, required []string) bool {
	for _, item := range required {
		if _, found := slices.BinarySearch(actual, item); !found {
			return false
		}
	}
	return true
}

func normalizeNames(policyID, kind string, values []string) ([]string, error) {
	normalized, err := normalize(kind, values)
	if err != nil {
		return nil, fmt.Errorf("construct authorization policy %q: %w", policyID, err)
	}
	return normalized, nil
}

func normalizeClaims(kind string, values []string) ([]string, error) {
	normalized, err := normalize(kind, values)
	if err != nil {
		return nil, fmt.Errorf("construct principal: %w", err)
	}
	return normalized, nil
}

func normalize(kind string, values []string) ([]string, error) {
	result := append([]string(nil), values...)
	for index, value := range result {
		if value == "" || strings.TrimSpace(value) != value {
			return nil, fmt.Errorf("%s %d must be non-empty and have no surrounding space", kind, index)
		}
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

func validateFrozenPolicy(policy Policy) error {
	if policy.definition.ID == "" || policy.definition.Module == "" {
		return errors.New("authorize: policy is invalid")
	}
	return nil
}
