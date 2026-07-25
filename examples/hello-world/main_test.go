package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunCheckConstructsGeneratedApplication(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	if err := run([]string{"-check"}, &stdout); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "Spice example ready: GET /users/{id}" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	if err := run([]string{"-unknown"}, &stdout); err == nil {
		t.Fatal("run() error = nil")
	}
}
