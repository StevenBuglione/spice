package security

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestAuthorizeDefinition(t *testing.T) {
	t.Parallel()
	if err := Authorize().Validate(); err != nil {
		t.Fatalf("Authorize() definition: %v", err)
	}
}

func TestAuthorizeHandler(t *testing.T) {
	t.Parallel()
	result, err := AuthorizeHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/StevenBuglione/spice/annotation/security",
		DescriptorSymbol:  "Authorize",
		CanonicalName:     "security.Authorize",
		Arguments: []sdk.InvocationArgument{
			{
				Name:  "authenticated",
				Kind:  sdk.KindBoolean,
				Value: json.RawMessage(`true`),
			},
			{
				Name:  "anyRoles",
				Kind:  sdk.KindList,
				Value: json.RawMessage(`["operator"]`),
			},
			{
				Name:  "allRoles",
				Kind:  sdk.KindList,
				Value: json.RawMessage(`["employee"]`),
			},
			{
				Name:  "allScopes",
				Kind:  sdk.KindList,
				Value: json.RawMessage(`["orders:read"]`),
			},
			{
				Name:  "expression",
				Kind:  sdk.KindString,
				Value: json.RawMessage(`"authenticated && hasRole(\"operator\")"`),
			},
		},
	})
	if err != nil || len(result.Contributions) != 1 {
		t.Fatalf("AuthorizeHandler() = %#v, %v", result, err)
	}
	policy := result.Contributions[0].Authorization
	if !policy.Authenticated || len(policy.AnyRoles) != 1 ||
		len(policy.AllRoles) != 1 || len(policy.AllScopes) != 1 ||
		policy.Expression == "" {
		t.Fatalf("authorization contribution = %#v", policy)
	}
	if _, err := AuthorizeHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "example.com/wrong",
		DescriptorSymbol:  "Authorize",
	}); err == nil {
		t.Fatal("AuthorizeHandler accepted a foreign descriptor")
	}
}
