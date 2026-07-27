package data

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestTransactionalDefinition(t *testing.T) {
	t.Parallel()
	if err := Transactional().Validate(); err != nil {
		t.Fatalf("Transactional() definition: %v", err)
	}
}

func TestTransactionalHandler(t *testing.T) {
	t.Parallel()
	result, err := TransactionalHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/StevenBuglione/spice/annotation/data",
		DescriptorSymbol:  "Transactional",
		CanonicalName:     "data.Transactional",
		Arguments: []sdk.InvocationArgument{
			{
				Name:  "isolation",
				Kind:  sdk.KindString,
				Value: json.RawMessage(`"serializable"`),
			},
			{
				Name:  "readOnly",
				Kind:  sdk.KindBoolean,
				Value: json.RawMessage(`true`),
			},
		},
	})
	if err != nil || len(result.Contributions) != 1 ||
		result.Contributions[0].Transaction.Isolation != "serializable" ||
		!result.Contributions[0].Transaction.ReadOnly {
		t.Fatalf("TransactionalHandler() = %#v, %v", result, err)
	}
	if _, err := TransactionalHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "example.com/wrong",
		DescriptorSymbol:  "Transactional",
	}); err == nil {
		t.Fatal("TransactionalHandler accepted a foreign descriptor")
	}
}
