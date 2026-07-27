package core

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestCoreDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{
		Application(),
		Bean(),
		Configuration(),
		Fallback(),
		Implements(),
		Order(),
		Primary(),
		Prototype(),
		Qualifier(),
		Repository(),
		RequestScope(),
		Service(),
		SessionScope(),
		Singleton(),
	} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}

func TestCoreHandlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		symbol    string
		canonical string
		handler   sdk.Handler
		arguments []sdk.InvocationArgument
		kind      sdk.ContributionKind
	}{
		{
			name:      "application",
			symbol:    "Application",
			canonical: "Application",
			handler:   ApplicationHandler,
			kind:      sdk.ContributionApplication,
		},
		{
			name:      "bean",
			symbol:    "Bean",
			canonical: "Bean",
			handler:   BeanHandler,
			kind:      sdk.ContributionProvider,
		},
		{
			name:      "configuration",
			symbol:    "Configuration",
			canonical: "Configuration",
			handler:   ConfigurationHandler,
			arguments: []sdk.InvocationArgument{{
				Name:  "prefix",
				Kind:  sdk.KindString,
				Value: json.RawMessage(`"commerce"`),
			}},
			kind: sdk.ContributionConfiguration,
		},
		{
			name:      "implements",
			symbol:    "Implements",
			canonical: "Implements",
			handler:   ImplementsHandler,
			arguments: []sdk.InvocationArgument{{
				Kind:       sdk.KindIdentifier,
				Positional: true,
				Value:      json.RawMessage(`"payments.Processor"`),
			}},
			kind: sdk.ContributionInterface,
		},
		{
			name:      "qualifier",
			symbol:    "Qualifier",
			canonical: "Qualifier",
			handler:   QualifierHandler,
			arguments: []sdk.InvocationArgument{{
				Kind:       sdk.KindString,
				Positional: true,
				Value:      json.RawMessage(`"stripe"`),
			}},
			kind: sdk.ContributionBeanMetadata,
		},
		{
			name:      "primary",
			symbol:    "Primary",
			canonical: "Primary",
			handler:   PrimaryHandler,
			kind:      sdk.ContributionBeanMetadata,
		},
		{
			name:      "fallback",
			symbol:    "Fallback",
			canonical: "Fallback",
			handler:   FallbackHandler,
			kind:      sdk.ContributionBeanMetadata,
		},
		{
			name:      "order",
			symbol:    "Order",
			canonical: "Order",
			handler:   OrderHandler,
			arguments: []sdk.InvocationArgument{{
				Kind:       sdk.KindInteger,
				Positional: true,
				Value:      json.RawMessage(`-10`),
			}},
			kind: sdk.ContributionBeanMetadata,
		},
		{
			name:      "singleton",
			symbol:    "Singleton",
			canonical: "Singleton",
			handler:   SingletonHandler,
			kind:      sdk.ContributionBeanMetadata,
		},
		{
			name:      "prototype",
			symbol:    "Prototype",
			canonical: "Prototype",
			handler:   PrototypeHandler,
			kind:      sdk.ContributionBeanMetadata,
		},
		{
			name:      "request scope",
			symbol:    "RequestScope",
			canonical: "RequestScope",
			handler:   RequestScopeHandler,
			kind:      sdk.ContributionBeanMetadata,
		},
		{
			name:      "session scope",
			symbol:    "SessionScope",
			canonical: "SessionScope",
			handler:   SessionScopeHandler,
			kind:      sdk.ContributionBeanMetadata,
		},
		{
			name:      "repository",
			symbol:    "Repository",
			canonical: "Repository",
			handler:   RepositoryHandler,
			kind:      sdk.ContributionStereotype,
		},
		{
			name:      "service",
			symbol:    "Service",
			canonical: "Service",
			handler:   ServiceHandler,
			kind:      sdk.ContributionStereotype,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result, err := test.handler(context.Background(), sdk.Invocation{
				DescriptorPackage: "github.com/StevenBuglione/spice/annotation/core",
				DescriptorSymbol:  test.symbol,
				CanonicalName:     test.canonical,
				Arguments:         test.arguments,
			})
			if err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if len(result.Contributions) != 1 ||
				result.Contributions[0].Kind != test.kind {
				t.Fatalf("handler result = %#v", result)
			}
			if _, err := test.handler(context.Background(), sdk.Invocation{
				DescriptorPackage: "example.com/wrong",
				DescriptorSymbol:  test.symbol,
			}); err == nil {
				t.Fatal("handler accepted a foreign descriptor")
			}
		})
	}
}
