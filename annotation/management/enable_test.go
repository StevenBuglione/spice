package management

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
)

func TestEnableDefinition(t *testing.T) {
	t.Parallel()
	if err := Enable().Validate(); err != nil {
		t.Fatalf("Enable() definition: %v", err)
	}
}

func TestManagementEnableHandler(t *testing.T) {
	t.Parallel()
	result, err := ManagementEnableHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/spice-framework/spice/annotation/management",
		DescriptorSymbol:  "Enable",
		CanonicalName:     "management.Enable",
		Arguments: []sdk.InvocationArgument{{
			Name:  "expose",
			Kind:  sdk.KindList,
			Value: json.RawMessage(`["health","metrics"]`),
		}, {
			Name:  "access",
			Kind:  sdk.KindString,
			Value: json.RawMessage(`"loopback"`),
		}},
	})
	if err != nil || len(result.Contributions) != 1 ||
		len(result.Contributions[0].Bootstrap.Options) != 2 {
		t.Fatalf("ManagementEnableHandler() = %#v, %v", result, err)
	}
	options := result.Contributions[0].Bootstrap.Options
	if options[1].Name != "access" ||
		options[1].Value.Kind != sdk.KindString ||
		options[1].Value.String != "loopback" {
		t.Fatalf("ManagementEnableHandler() options = %#v", options)
	}
	if _, err := ManagementEnableHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "example.com/wrong",
		DescriptorSymbol:  "Enable",
	}); err == nil {
		t.Fatal("ManagementEnableHandler accepted a foreign descriptor")
	}
}
