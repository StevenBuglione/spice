package protocol

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestServeDispatchesTypedLifecycle(t *testing.T) {
	var input bytes.Buffer
	writeServerRequest(t, &input, 1, "initialize", InitializeParams{
		Protocol: VersionV1Alpha2,
	})
	writeServerRequest(t, &input, 2, "describe", DescribeParams{})
	writeServerRequest(t, &input, 3, "analyze", AnalyzeParams{
		Descriptor: sdk.Symbol{
			Package: "example.com/annotation",
			Name:    "Fixture",
		},
	})
	writeServerRequest(t, &input, 4, "shutdown", ShutdownParams{})
	var output bytes.Buffer
	tool := &fixtureProtocolTool{}
	if err := Serve(
		context.Background(),
		&input,
		&output,
		tool,
	); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	reader := bufio.NewReader(&output)
	for id := uint64(1); id <= 4; id++ {
		var response Response
		if err := ReadMessage(reader, &response); err != nil {
			t.Fatalf("ReadMessage(%d) error = %v", id, err)
		}
		if response.ID != id || response.Error != nil {
			t.Fatalf("response %d = %#v", id, response)
		}
	}
	if tool.calls != "initialize,describe,analyze,shutdown" {
		t.Fatalf("calls = %q", tool.calls)
	}
}

func TestServeReturnsProtocolErrorsAndContainsPanics(t *testing.T) {
	var input bytes.Buffer
	writeServerRequest(t, &input, 1, "missing", struct{}{})
	writeServerRequest(t, &input, 2, "analyze", AnalyzeParams{
		Descriptor: sdk.Symbol{
			Package: "example.com/annotation",
			Name:    "Panic",
		},
	})
	writeServerRequest(t, &input, 3, "shutdown", ShutdownParams{})
	var output bytes.Buffer
	if err := Serve(
		context.Background(),
		&input,
		&output,
		&fixtureProtocolTool{},
	); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	reader := bufio.NewReader(&output)
	var messages []string
	for range 3 {
		var response Response
		if err := ReadMessage(reader, &response); err != nil {
			t.Fatalf("ReadMessage() error = %v", err)
		}
		if response.Error != nil {
			messages = append(messages, response.Error.Message)
		}
	}
	joined := strings.Join(messages, "\n")
	if !strings.Contains(joined, "was not found") ||
		!strings.Contains(joined, "panic") {
		t.Fatalf("errors = %q", joined)
	}
}

func TestServeRejectsMissingDependenciesAndCancellation(t *testing.T) {
	if err := Serve(
		context.Background(),
		nil,
		&bytes.Buffer{},
		&fixtureProtocolTool{},
	); err == nil {
		t.Fatal("Serve(nil reader) error = nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Serve(
		ctx,
		bytes.NewReader(nil),
		&bytes.Buffer{},
		&fixtureProtocolTool{},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("Serve(cancelled) error = %v", err)
	}
}

type fixtureProtocolTool struct {
	calls string
}

func (tool *fixtureProtocolTool) Initialize(
	_ context.Context,
	_ InitializeParams,
) (InitializeResult, error) {
	tool.record("initialize")
	return InitializeResult{
		Protocol:   VersionV1Alpha2,
		ToolPath:   "example.com/tool",
		ModulePath: "example.com",
	}, nil
}

func (tool *fixtureProtocolTool) Describe(
	_ context.Context,
	_ DescribeParams,
) (DescribeResult, error) {
	tool.record("describe")
	return DescribeResult{
		DescriptorPackages: []string{"example.com/annotation"},
		Handlers: []Handler{{
			Descriptor: sdk.Symbol{
				Package: "example.com/annotation",
				Name:    "Fixture",
			},
		}},
	}, nil
}

func (tool *fixtureProtocolTool) Analyze(
	_ context.Context,
	params AnalyzeParams,
) (AnalyzeResult, error) {
	tool.record("analyze")
	if params.Descriptor.Name == "Panic" {
		panic("fixture panic")
	}
	return AnalyzeResult{}, nil
}

func (tool *fixtureProtocolTool) Shutdown(
	_ context.Context,
	_ ShutdownParams,
) error {
	tool.record("shutdown")
	return nil
}

func (tool *fixtureProtocolTool) record(name string) {
	if tool.calls != "" {
		tool.calls += ","
	}
	tool.calls += name
}

func writeServerRequest(
	t *testing.T,
	writer *bytes.Buffer,
	id uint64,
	method string,
	params any,
) {
	t.Helper()
	content, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := WriteMessage(writer, Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  content,
	}); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}
}
