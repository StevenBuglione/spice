package web

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestWebDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{
		Controller(),
		Get(),
		Post(),
	} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}

func TestWebHandlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		symbol    string
		handler   sdk.Handler
		arguments []sdk.InvocationArgument
		kind      sdk.ContributionKind
	}{
		{
			symbol:  "Controller",
			handler: ControllerHandler,
			arguments: []sdk.InvocationArgument{{
				Name:  "prefix",
				Kind:  sdk.KindString,
				Value: json.RawMessage(`"/orders"`),
			}},
			kind: sdk.ContributionController,
		},
		{
			symbol:  "Get",
			handler: GetHandler,
			arguments: []sdk.InvocationArgument{{
				Kind:       sdk.KindString,
				Positional: true,
				Value:      json.RawMessage(`"/orders/{id}"`),
			}},
			kind: sdk.ContributionRoute,
		},
		{
			symbol:  "Post",
			handler: PostHandler,
			arguments: []sdk.InvocationArgument{{
				Name:  "path",
				Kind:  sdk.KindString,
				Value: json.RawMessage(`"/orders"`),
			}},
			kind: sdk.ContributionRoute,
		},
	}
	for _, test := range tests {
		t.Run(test.symbol, func(t *testing.T) {
			t.Parallel()
			result, err := test.handler(context.Background(), sdk.Invocation{
				DescriptorPackage: "github.com/StevenBuglione/spice/annotation/web",
				DescriptorSymbol:  test.symbol,
				CanonicalName:     "web." + test.symbol,
				Arguments:         test.arguments,
			})
			if err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if len(result.Contributions) == 0 ||
				result.Contributions[0].Kind != test.kind {
				t.Fatalf("handler result = %#v", result)
			}
			if test.symbol == "Controller" &&
				(len(result.Contributions) != 2 ||
					result.Contributions[1].Kind != sdk.ContributionStereotype) {
				t.Fatalf("controller stereotype = %#v", result)
			}
			if _, err := test.handler(context.Background(), sdk.Invocation{
				DescriptorPackage: "example.com/wrong",
				DescriptorSymbol:  test.symbol,
			}); err == nil {
				t.Fatal("handler accepted a foreign descriptor")
			}
		})
	}
	if _, err := GetHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/StevenBuglione/spice/annotation/web",
		DescriptorSymbol:  "Get",
		CanonicalName:     "web.Get",
		Arguments: []sdk.InvocationArgument{{
			Kind:       sdk.KindString,
			Positional: true,
			Value:      json.RawMessage(`"relative"`),
		}},
	}); err == nil {
		t.Fatal("GetHandler accepted a relative route")
	}
}
