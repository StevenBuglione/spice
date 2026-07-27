package handler

import (
	"context"
	"errors"

	"github.com/StevenBuglione/spice/annotation/sdk"
	"github.com/StevenBuglione/spice/annotation/sdk/protocol"
)

// FactoryHandler implements fixture.Factory through the public contribution
// protocol.
func FactoryHandler(
	_ context.Context,
	invocation protocol.Invocation,
) (protocol.AnalyzeResult, error) {
	if invocation.DescriptorPackage !=
		"example.com/spice-annotation-fixture/annotation/wiring" ||
		invocation.DescriptorSymbol != "Factory" {
		return protocol.AnalyzeResult{}, errors.New(
			"factory handler received an unexpected descriptor",
		)
	}
	return encode(sdk.Contribution{
		Kind:     sdk.ContributionProvider,
		Provider: &sdk.ProviderContribution{},
	})
}

func encode(
	contribution sdk.Contribution,
) (protocol.AnalyzeResult, error) {
	value, err := protocol.EncodeContribution(contribution)
	if err != nil {
		return protocol.AnalyzeResult{}, err
	}
	return protocol.AnalyzeResult{
		Contributions: []protocol.Contribution{value},
	}, nil
}
