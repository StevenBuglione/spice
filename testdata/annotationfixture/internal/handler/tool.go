// Package handler implements the fixture's public annotation protocol tool
// without importing any Spice compiler or CLI package.
package handler

import (
	"context"
	"errors"

	"github.com/StevenBuglione/spice/annotation/sdk"
	"github.com/StevenBuglione/spice/annotation/sdk/protocol"
)

const (
	toolPath   = "example.com/spice-annotation-fixture/cmd/spice-annotations"
	modulePath = "example.com/spice-annotation-fixture"
)

// Tool is one isolated fixture annotation protocol implementation.
type Tool struct{}

// Initialize validates protocol and Go tool identities.
func (Tool) Initialize(
	_ context.Context,
	params protocol.InitializeParams,
) (protocol.InitializeResult, error) {
	if params.Protocol != sdk.ProtocolV1Alpha1 ||
		params.ToolPath != toolPath {
		return protocol.InitializeResult{}, errors.New(
			"fixture annotation tool identity does not match",
		)
	}
	return protocol.InitializeResult{
		Protocol:      sdk.ProtocolV1Alpha1,
		ToolPath:      toolPath,
		ModulePath:    modulePath,
		ModuleVersion: "v0.0.0",
	}, nil
}

// Describe returns every fixture handler and its real source symbol.
func (Tool) Describe(
	context.Context,
	protocol.DescribeParams,
) (protocol.DescribeResult, error) {
	return protocol.DescribeResult{Handlers: []protocol.Handler{
		{
			ID:           "fixture/factory",
			Capabilities: []string{string(sdk.ContributionProvider)},
			Source: sdk.Symbol{
				Package: modulePath + "/internal/handler",
				Name:    "FactoryHandler",
			},
		},
		{
			ID:           "fixture/policy",
			Capabilities: []string{string(sdk.ContributionStereotype)},
			Source: sdk.Symbol{
				Package: modulePath + "/internal/handler",
				Name:    "PolicyHandler",
			},
		},
	}}, nil
}

// Analyze dispatches only the handlers declared by Describe.
func (Tool) Analyze(
	ctx context.Context,
	params protocol.AnalyzeParams,
) (protocol.AnalyzeResult, error) {
	switch params.Handler {
	case "fixture/factory":
		return FactoryHandler(ctx, params.Invocation)
	case "fixture/policy":
		return PolicyHandler(ctx, params.Invocation)
	default:
		return protocol.AnalyzeResult{}, errors.New(
			"fixture annotation handler is not declared",
		)
	}
}

// Shutdown releases no resources because Tool owns no globals.
func (Tool) Shutdown(
	context.Context,
	protocol.ShutdownParams,
) error {
	return nil
}
