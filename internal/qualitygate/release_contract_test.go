package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckVerifyReleaseTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{name: "exact target", content: "verify:\n\tgo run ./internal/qualitygate -mode=verify\n\nverify-release:\n\tgo run ./internal/qualitygate -mode=verify-release\n"},
		{name: "missing target", content: "verify:\n\tgo run ./internal/qualitygate -mode=verify\n", wantErr: true},
		{name: "reduced target", content: "verify-release:\n\tgo run ./internal/qualitygate -mode=check\n", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			err := checkVerifyReleaseTarget(root)
			if (err != nil) != test.wantErr {
				t.Fatalf("checkVerifyReleaseTarget() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestCheckReleaseWorkflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(string) string
		wantErr bool
	}{
		{name: "exact contract", mutate: func(content string) string { return content }},
		{name: "wrong revision", mutate: func(content string) string {
			return strings.Replace(content, releaseWorkflowRevision, strings.Repeat("0", 40), 1)
		}, wantErr: true},
		{name: "wrong module", mutate: func(content string) string {
			return strings.Replace(content, modulePath, modulePath+"-wrong", 1)
		}, wantErr: true},
		{name: "repository wide write permission", mutate: func(content string) string {
			return strings.Replace(content, "permissions: {}", "permissions:\n  contents: write", 1)
		}, wantErr: true},
		{name: "missing job permission", mutate: func(content string) string {
			return strings.Replace(content, "    permissions:\n      contents: write\n", "", 1)
		}, wantErr: true},
		{name: "missing signing secret", mutate: func(content string) string {
			return strings.Replace(content, "    secrets:\n      SPICE_LIBRARY_RELEASE_SIGNING_KEY: ${{ secrets.SPICE_LIBRARY_RELEASE_SIGNING_KEY }}\n", "", 1)
		}, wantErr: true},
		{name: "inherited secrets", mutate: func(content string) string {
			return strings.Replace(content, "    secrets:\n      SPICE_LIBRARY_RELEASE_SIGNING_KEY: ${{ secrets.SPICE_LIBRARY_RELEASE_SIGNING_KEY }}\n", "    secrets: inherit\n", 1)
		}, wantErr: true},
		{name: "additional secret", mutate: func(content string) string {
			return strings.Replace(content, "      SPICE_LIBRARY_RELEASE_SIGNING_KEY: ${{ secrets.SPICE_LIBRARY_RELEASE_SIGNING_KEY }}\n", "      SPICE_LIBRARY_RELEASE_SIGNING_KEY: ${{ secrets.SPICE_LIBRARY_RELEASE_SIGNING_KEY }}\n      UNRELATED_SECRET: ${{ secrets.UNRELATED_SECRET }}\n", 1)
		}, wantErr: true},
		{name: "wrong secret source", mutate: func(content string) string {
			return strings.Replace(content, "${{ secrets.SPICE_LIBRARY_RELEASE_SIGNING_KEY }}", "${{ secrets.OTHER_KEY }}", 1)
		}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, ".github", "workflows", "release.yml")
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(test.mutate(expectedReleaseWorkflow(modulePath))), 0o600); err != nil {
				t.Fatal(err)
			}
			err := checkReleaseWorkflow(root)
			if (err != nil) != test.wantErr {
				t.Fatalf("checkReleaseWorkflow() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestCheckReleasePublicKey(t *testing.T) {
	t.Parallel()
	repository, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(repository, "security", "release", "ed25519-public.pem"))
	if err != nil {
		t.Fatal(err)
	}

	differentDER, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		t.Fatal(err)
	}
	differentKey := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: differentDER})
	tests := []struct {
		name    string
		content []byte
		wantErr bool
	}{
		{name: "exact trust anchor", content: content},
		{name: "malformed PEM", content: []byte("not a public key\n"), wantErr: true},
		{name: "different valid Ed25519 key", content: differentKey, wantErr: true},
		{name: "trailing PEM", content: append(append([]byte(nil), content...), content...), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "security", "release", "ed25519-public.pem")
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, test.content, 0o600); err != nil {
				t.Fatal(err)
			}
			err := checkReleasePublicKey(root)
			if (err != nil) != test.wantErr {
				t.Fatalf("checkReleasePublicKey() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
