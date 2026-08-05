package protocol

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
)

func TestContributionRoundTrip(t *testing.T) {
	t.Parallel()
	input := sdk.Contribution{
		Kind: sdk.ContributionRoute,
		Route: &sdk.RouteContribution{
			Method: "GET",
			Path:   "/orders/{id}",
		},
	}
	wire, err := EncodeContribution(input)
	if err != nil {
		t.Fatalf("EncodeContribution() error = %v", err)
	}
	decoded, err := DecodeContribution(wire)
	if err != nil {
		t.Fatalf("DecodeContribution() error = %v", err)
	}
	if decoded.Route == nil ||
		decoded.Route.Method != http.MethodGet ||
		decoded.Route.Path != "/orders/{id}" {
		t.Fatalf("DecodeContribution() = %+v", decoded)
	}
}

func TestEveryContributionKindRoundTrips(t *testing.T) {
	t.Parallel()
	values := []sdk.Contribution{
		{Kind: sdk.ContributionApplication, Application: &sdk.ApplicationContribution{}},
		{Kind: sdk.ContributionStereotype, Stereotype: &sdk.StereotypeContribution{Role: "service"}},
		{Kind: sdk.ContributionInterface, Interface: &sdk.InterfaceBindingContribution{Interfaces: []string{"payments.Processor"}}},
		{Kind: sdk.ContributionProvider, Provider: &sdk.ProviderContribution{}},
		{
			Kind: sdk.ContributionBeanMetadata,
			BeanMetadata: &sdk.BeanMetadataContribution{
				Qualifiers: []string{"stripe"},
				Primary:    true,
				Scope:      sdk.BeanScopeRequest,
			},
		},
		{Kind: sdk.ContributionConfiguration, Configuration: &sdk.ConfigurationContribution{Prefix: "server"}},
		{Kind: sdk.ContributionController, Controller: &sdk.ControllerContribution{Prefix: "/api"}},
		{Kind: sdk.ContributionRoute, Route: &sdk.RouteContribution{Method: http.MethodGet, Path: "/"}},
		{Kind: sdk.ContributionModule, Module: &sdk.ModuleContribution{AllowedDependencies: []string{"example.com/api"}}},
		{Kind: sdk.ContributionNamedInterface, NamedInterface: &sdk.NamedInterfaceContribution{Name: "api"}},
		{Kind: sdk.ContributionLifecycle, Lifecycle: &sdk.LifecycleContribution{Phase: sdk.LifecycleStop}},
		{Kind: sdk.ContributionBootstrap, Bootstrap: &sdk.BootstrapContribution{Capability: "management"}},
		{Kind: sdk.ContributionSchedule, Schedule: &sdk.ScheduleContribution{Delay: "1m"}},
		{Kind: sdk.ContributionAsync, Async: &sdk.AsyncContribution{}},
		{Kind: sdk.ContributionTransaction, Transaction: &sdk.TransactionContribution{Isolation: "serializable"}},
		{Kind: sdk.ContributionEventTopic, EventTopic: &sdk.EventTopicContribution{}},
		{Kind: sdk.ContributionEventListener, EventListener: &sdk.EventListenerContribution{Order: 2}},
		{Kind: sdk.ContributionCache, Cache: &sdk.CacheContribution{Name: "orders"}},
		{Kind: sdk.ContributionAuthorization, Authorization: &sdk.AuthorizationContribution{AnyRoles: []string{"operator"}}},
		{Kind: sdk.ContributionGeneratedFile, GeneratedFile: &sdk.GeneratedFileContribution{Path: "generated/feature.go", Content: "package generated\n"}},
	}
	for _, value := range values {
		wire, err := EncodeContribution(value)
		if err != nil {
			t.Fatalf("EncodeContribution(%q) error = %v", value.Kind, err)
		}
		decoded, err := DecodeContribution(wire)
		if err != nil {
			t.Fatalf("DecodeContribution(%q) error = %v", value.Kind, err)
		}
		if decoded.Kind != value.Kind {
			t.Fatalf(
				"DecodeContribution() kind = %q, want %q",
				decoded.Kind,
				value.Kind,
			)
		}
	}
}

func TestDecodeContributionRejectsUnknownAndTrailingJSON(t *testing.T) {
	t.Parallel()
	tests := []Contribution{
		{
			Kind: sdk.ContributionRoute,
			Value: json.RawMessage(
				`{"method":"GET","path":"/","magic":true}`,
			),
		},
		{
			Kind:  sdk.ContributionApplication,
			Value: json.RawMessage(`{} {}`),
		},
		{
			Kind:  sdk.ContributionKind("unknown"),
			Value: json.RawMessage(`{}`),
		},
	}
	for _, input := range tests {
		if _, err := DecodeContribution(input); err == nil {
			t.Fatalf(
				"DecodeContribution(%s) error = nil, want failure",
				input.Kind,
			)
		}
	}
}
