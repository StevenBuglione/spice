package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBestMaturityPrefixUsesMostSpecificRule(t *testing.T) {
	t.Parallel()
	rules := []maturityClassification{
		{Prefix: "annotation"},
		{Prefix: "annotation/sdk"},
	}
	got, ok := bestMaturityPrefix("annotation/sdk/protocol", rules)
	if !ok || got != "annotation/sdk" {
		t.Fatalf("bestMaturityPrefix() = %q, %v", got, ok)
	}
}

func TestAggregateCoverageWeightsStatements(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "coverage.out")
	content := "mode: atomic\n" +
		modulePath + "/bean/bean.go:1.1,2.1 3 1\n" +
		modulePath + "/web/web.go:1.1,2.1 1 0\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := aggregateCoverage(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != 75 {
		t.Fatalf("aggregateCoverage() = %v, want 75", got)
	}
}

func TestValidateMaturityClassificationsRejectsUnusedRule(t *testing.T) {
	t.Parallel()
	rules := []maturityClassification{
		{Prefix: "bean", Maturity: "preview-stable", Reason: "stable contract"},
		{Prefix: "web", Maturity: "experimental", Reason: "evolving contract"},
	}
	err := validateMaturityClassifications(rules, []string{modulePath + "/bean"})
	if err == nil {
		t.Fatal("validateMaturityClassifications() succeeded with an unused rule")
	}
}

func TestValidateImportsEnforcesToolchainBoundary(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tests := []struct {
		name      string
		source    string
		wantError string
	}{
		{
			name: "standalone toolchain root import",
			source: `package bean

import _ "github.com/spice-framework/toolchain"
`,
			wantError: officialToolchainPath,
		},
		{
			name: "standalone toolchain package import",
			source: `package bean

import _ "github.com/spice-framework/toolchain/compiler/service"
`,
			wantError: officialToolchainPath + "/compiler/service",
		},
		{
			name: "descriptor tool path metadata",
			source: `package coretool

const Path = "github.com/spice-framework/toolchain/cmd/spice-annotation-core"
`,
		},
		{
			name: "standard library import",
			source: `package bean

import "context"
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(root, "bean", "service.go")
			parsed, err := parser.ParseFile(
				token.NewFileSet(),
				path,
				test.source,
				parser.ImportsOnly,
			)
			if err != nil {
				t.Fatal(err)
			}
			err = validateImports(root, path, parsed)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("validateImports() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("validateImports() error = %v, want text %q", err, test.wantError)
			}
		})
	}
}
