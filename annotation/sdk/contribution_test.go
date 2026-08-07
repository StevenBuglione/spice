package sdk

import "testing"

func TestContributionValidateAndClone(t *testing.T) {
	t.Parallel()
	original := Contribution{
		Kind: ContributionAuthorization,
		Authorization: &AuthorizationContribution{
			Authenticated: true,
			AnyRoles:      []string{"operator"},
			AllScopes:     []string{"orders.read"},
			Expression:    `authenticated && hasRole("operator")`,
		},
	}
	if err := original.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	cloned := original.Clone()
	cloned.Authorization.AnyRoles[0] = "changed"
	if original.Authorization.AnyRoles[0] != "operator" {
		t.Fatal("Clone() retained an authorization slice alias")
	}
}

func TestAuthorizationContributionRejectsExpressionWhitespace(t *testing.T) {
	t.Parallel()
	contribution := Contribution{
		Kind: ContributionAuthorization,
		Authorization: &AuthorizationContribution{
			Expression: " authenticated",
		},
	}
	if err := contribution.Validate(); err == nil {
		t.Fatal("Validate() accepted expression whitespace")
	}
}

func TestBeanMetadataContributionClonesOwnedValues(t *testing.T) {
	t.Parallel()
	order := int64(-10)
	original := Contribution{
		Kind: ContributionBeanMetadata,
		BeanMetadata: &BeanMetadataContribution{
			Qualifiers: []string{"stripe"},
			Order:      &order,
			Scope:      BeanScopeRequest,
		},
	}
	cloned := original.Clone()
	cloned.BeanMetadata.Qualifiers[0] = "changed"
	*cloned.BeanMetadata.Order = 20
	if original.BeanMetadata.Qualifiers[0] != "stripe" ||
		*original.BeanMetadata.Order != -10 {
		t.Fatal("Clone() retained bean metadata aliases")
	}
}

func TestContributionRejectsMalformedValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value Contribution
	}{
		{
			name:  "missing payload",
			value: Contribution{Kind: ContributionApplication},
		},
		{
			name: "ambiguous payload",
			value: Contribution{
				Kind:        ContributionApplication,
				Application: &ApplicationContribution{},
				Provider:    &ProviderContribution{},
			},
		},
		{
			name: "kind mismatch",
			value: Contribution{
				Kind:     ContributionProvider,
				Provider: nil,
				Route:    &RouteContribution{Method: "GET", Path: "/"},
			},
		},
		{
			name: "relative route",
			value: Contribution{
				Kind: ContributionRoute,
				Route: &RouteContribution{
					Method: "GET",
					Path:   "orders",
				},
			},
		},
		{
			name: "unsafe generated path",
			value: Contribution{
				Kind: ContributionGeneratedFile,
				GeneratedFile: &GeneratedFileContribution{
					Path:    "../manual.go",
					Content: "package generated",
				},
			},
		},
		{
			name: "empty interface binding",
			value: Contribution{
				Kind:      ContributionInterface,
				Interface: &InterfaceBindingContribution{},
			},
		},
		{
			name: "duplicate interface binding",
			value: Contribution{
				Kind: ContributionInterface,
				Interface: &InterfaceBindingContribution{
					Interfaces: []string{"api.Reader", "api.Reader"},
				},
			},
		},
		{
			name: "untrimmed stereotype constructor",
			value: Contribution{
				Kind: ContributionStereotype,
				Stereotype: &StereotypeContribution{
					Role:        "service",
					Constructor: " NewService",
				},
			},
		},
		{
			name: "primary fallback conflict",
			value: Contribution{
				Kind: ContributionBeanMetadata,
				BeanMetadata: &BeanMetadataContribution{
					Primary:  true,
					Fallback: true,
				},
			},
		},
		{
			name: "duplicate qualifier",
			value: Contribution{
				Kind: ContributionBeanMetadata,
				BeanMetadata: &BeanMetadataContribution{
					Qualifiers: []string{"stripe", "stripe"},
				},
			},
		},
		{
			name: "unsupported scope",
			value: Contribution{
				Kind: ContributionBeanMetadata,
				BeanMetadata: &BeanMetadataContribution{
					Scope: BeanScope("thread"),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.value.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want failure")
			}
		})
	}
}

func TestEveryContributionKindValidatesAndClones(t *testing.T) {
	t.Parallel()
	for _, contribution := range validContributions() {
		if err := contribution.Validate(); err != nil {
			t.Fatalf("Validate(%q) error = %v", contribution.Kind, err)
		}
		cloned := contribution.Clone()
		if err := cloned.Validate(); err != nil {
			t.Fatalf("Clone(%q) error = %v", contribution.Kind, err)
		}
	}
}

func validContributions() []Contribution {
	return []Contribution{
		{
			Kind:        ContributionApplication,
			Application: &ApplicationContribution{},
		},
		{
			Kind:       ContributionStereotype,
			Stereotype: &StereotypeContribution{Role: "service"},
		},
		{
			Kind: ContributionInterface,
			Interface: &InterfaceBindingContribution{
				Interfaces: []string{
					"health.Checker",
					"payments.Processor",
				},
			},
		},
		{
			Kind:     ContributionProvider,
			Provider: &ProviderContribution{},
		},
		{
			Kind: ContributionBeanMetadata,
			BeanMetadata: &BeanMetadataContribution{
				Qualifiers: []string{"stripe"},
				Primary:    true,
				Order:      int64Pointer(-10),
				Scope:      BeanScopeRequest,
			},
		},
		{
			Kind: ContributionConfiguration,
			Configuration: &ConfigurationContribution{
				Prefix: "server.http",
			},
		},
		{
			Kind: ContributionEnum,
			Enum: &EnumContribution{},
		},
		{
			Kind:       ContributionController,
			Controller: &ControllerContribution{Prefix: "/api"},
		},
		{
			Kind: ContributionRoute,
			Route: &RouteContribution{
				Method: "GET",
				Path:   "/orders",
			},
		},
		{
			Kind: ContributionModule,
			Module: &ModuleContribution{
				AllowedDependencies: []string{"example.com/inventory"},
			},
		},
		{
			Kind: ContributionNamedInterface,
			NamedInterface: &NamedInterfaceContribution{
				Name: "api",
			},
		},
		{
			Kind: ContributionLifecycle,
			Lifecycle: &LifecycleContribution{
				Phase: LifecycleStart,
			},
		},
		{
			Kind: ContributionBootstrap,
			Bootstrap: &BootstrapContribution{
				Capability: "management",
				Options: []BootstrapOption{{
					Name: "expose",
					Value: ContributionValue{
						Kind: KindList,
						List: []ContributionValue{
							{Kind: KindString, String: "health"},
							{Kind: KindInteger, Integer: 2},
							{Kind: KindBoolean, Boolean: true},
							{
								Kind:       KindIdentifier,
								Identifier: "ready",
							},
						},
					},
				}},
			},
		},
		{
			Kind: ContributionSchedule,
			Schedule: &ScheduleContribution{
				Delay: "5m",
			},
		},
		{
			Kind:  ContributionAsync,
			Async: &AsyncContribution{},
		},
		{
			Kind: ContributionTransaction,
			Transaction: &TransactionContribution{
				Isolation: "serializable",
				ReadOnly:  true,
			},
		},
		{
			Kind:       ContributionEventTopic,
			EventTopic: &EventTopicContribution{},
		},
		{
			Kind: ContributionEventListener,
			EventListener: &EventListenerContribution{
				Order: 3,
			},
		},
		{
			Kind:  ContributionCache,
			Cache: &CacheContribution{Name: "orders.by-id"},
		},
		{
			Kind: ContributionAuthorization,
			Authorization: &AuthorizationContribution{
				Authenticated: true,
				AnyRoles:      []string{"operator"},
				AllRoles:      []string{"member"},
				AllScopes:     []string{"orders.read"},
				Expression:    `authenticated && hasRole("operator")`,
			},
		},
		{
			Kind: ContributionGeneratedFile,
			GeneratedFile: &GeneratedFileContribution{
				Path:    "internal/spicegen/plugin/feature.go",
				Content: "package plugin\n",
			},
		},
	}
}

func int64Pointer(value int64) *int64 {
	return new(value)
}
