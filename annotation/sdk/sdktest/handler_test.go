package sdktest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
	"github.com/spice-framework/spice/annotation/sdk/sdktest"
)

func TestRunHandlerCasesValidatesPublicSDKHandlers(t *testing.T) {
	t.Parallel()

	definition := testDefinition()
	base := sdktest.Invocation(
		"example.com/starter/annotation",
		"Feature",
		"starter.Feature",
		sdk.Declaration{
			Target:      sdk.TargetType,
			SymbolID:    "type:Service",
			Name:        "Service",
			PackagePath: "example.com/app",
		},
	)
	withRole := base
	withRole.Arguments = []sdk.InvocationArgument{
		sdktest.StringArgument("role", "worker", false),
	}
	invalid := base
	invalid.DescriptorSymbol = "Other"

	sdktest.RunHandlerCases(
		t,
		definition,
		sdktest.HandlerCase{
			Name:       "contribution",
			Invocation: withRole,
			WantKinds: []sdk.ContributionKind{
				sdk.ContributionStereotype,
			},
			Check: func(t *testing.T, result sdk.Result) {
				t.Helper()
				if result.Contributions[0].Stereotype.Role != "worker" {
					t.Fatalf(
						"role = %q",
						result.Contributions[0].Stereotype.Role,
					)
				}
			},
		},
		sdktest.HandlerCase{
			Name:              "descriptor mismatch",
			Invocation:        invalid,
			WantErrorContains: "received descriptor",
		},
	)
}

func TestRunHandlerCasesExercisesCancellation(t *testing.T) {
	t.Parallel()

	definition := testDefinition()
	definition.Implementation.Handler = func(
		ctx context.Context,
		_ sdk.Invocation,
	) (sdk.Result, error) {
		<-ctx.Done()
		return sdk.Result{}, context.Cause(ctx)
	}
	sdktest.RunHandlerCases(
		t,
		definition,
		sdktest.HandlerCase{
			Name:              "canceled",
			Canceled:          true,
			Invocation:        validInvocation(),
			WantErrorContains: context.Canceled.Error(),
		},
	)
}

func testDefinition() sdk.Definition {
	return sdk.Definition{
		Name:    "starter.Feature",
		Summary: "Classifies one third-party starter type.",
		Targets: []sdk.Target{sdk.TargetType},
		Arguments: []sdk.Argument{{
			Name:        "role",
			Kinds:       []sdk.Kind{sdk.KindString},
			Description: "Generated architecture role.",
		}},
		Examples: []sdk.Example{{
			Title: "Feature",
			Code:  "// @Feature(role=\"worker\")\ntype Service struct{}",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "1.0.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "example.com/starter/cmd/spice-annotations",
			Handler:  testHandler,
			Protocol: sdk.ProtocolV1Alpha2,
		},
	}
}

func testHandler(
	_ context.Context,
	invocation sdk.Invocation,
) (sdk.Result, error) {
	if err := invocation.RequireDescriptor(
		"example.com/starter/annotation",
		"Feature",
	); err != nil {
		return sdk.Result{}, err
	}
	arguments, err := sdk.BindArguments(invocation, "", "role")
	if err != nil {
		return sdk.Result{}, err
	}
	role, err := arguments.String("role", true)
	if err != nil {
		return sdk.Result{}, err
	}
	if role == "" {
		return sdk.Result{}, errors.New("role is empty")
	}
	return sdk.OneContribution(sdk.Contribution{
		Kind: sdk.ContributionStereotype,
		Stereotype: &sdk.StereotypeContribution{
			Role: role,
		},
	})
}

func validInvocation() sdk.Invocation {
	invocation := sdktest.Invocation(
		"example.com/starter/annotation",
		"Feature",
		"starter.Feature",
		sdk.Declaration{
			Target:      sdk.TargetType,
			SymbolID:    "type:Service",
			Name:        "Service",
			PackagePath: "example.com/app",
		},
	)
	invocation.Arguments = []sdk.InvocationArgument{
		sdktest.StringArgument("role", "worker", false),
	}
	return invocation
}
