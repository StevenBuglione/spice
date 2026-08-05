package security_test

import (
	"context"
	"fmt"

	"github.com/spice-framework/spice/security"
)

func ExampleAuthorizer() {
	policy, err := security.NewPolicy(security.PolicySpec{
		Definition: security.Definition{
			ID:     "orders.read",
			Module: "example.com/shop/orders",
		},
		AllScopes: []string{"orders:read"},
	})
	principal, principalErr := security.NewPrincipal(
		"user-1",
		"https://issuer.example",
		nil,
		[]string{"orders:read"},
	)
	if err != nil || principalErr != nil {
		fmt.Printf("construct: %v %v\n", err, principalErr)
		return
	}
	ctx, err := security.WithPrincipal(context.Background(), principal)
	authorizer, authorizerErr := security.NewAuthorizer()
	if err == nil && authorizerErr == nil {
		err = authorizer.Authorize(ctx, policy)
	}
	fmt.Printf("authorized=%v\n", err == nil)
	// Output:
	// authorized=true
}
