package handler

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/StevenBuglione/spice/annotation/sdk"
	"github.com/StevenBuglione/spice/annotation/sdk/protocol"
)

// PolicyHandler implements fixture.Policy and demonstrates plugin-owned
// diagnostics.
func PolicyHandler(
	_ context.Context,
	invocation protocol.Invocation,
) (protocol.AnalyzeResult, error) {
	mode, err := policyMode(invocation.Arguments)
	if err != nil {
		return protocol.AnalyzeResult{}, err
	}
	if mode == "deny" {
		return protocol.AnalyzeResult{
			Diagnostics: []protocol.Diagnostic{{
				Code:     "policy-denied",
				Severity: "error",
				Message:  "fixture policy deliberately denied this declaration",
			}},
		}, nil
	}
	return encode(sdk.Contribution{
		Kind: sdk.ContributionStereotype,
		Stereotype: &sdk.StereotypeContribution{
			Role: "fixture-policy",
		},
	})
}

func policyMode(arguments []protocol.Argument) (string, error) {
	if len(arguments) == 0 {
		return "strict", nil
	}
	if len(arguments) != 1 || arguments[0].Name != "mode" {
		return "", errors.New("policy handler requires only the mode argument")
	}
	var mode string
	if err := json.Unmarshal(arguments[0].Value, &mode); err != nil {
		return "", errors.New("policy handler mode must be a string")
	}
	return mode, nil
}
