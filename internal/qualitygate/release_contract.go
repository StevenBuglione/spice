package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	releaseWorkflowRevision     = "9ae80e32f64b29697acd9ebe629468850b4ae9f2"
	releasePublicKeyFingerprint = "a7d12fc21024a11f0472887a37c731697a0aa2c2f6b84ff3afef6d47563422f1"
)

func checkReleaseContract(root string) error {
	if err := checkVerifyReleaseTarget(root); err != nil {
		return err
	}
	if err := checkReleaseWorkflow(root); err != nil {
		return err
	}
	return checkReleasePublicKey(root)
}

func checkVerifyReleaseTarget(root string) error {
	content, err := os.ReadFile(filepath.Join(root, "Makefile")) // #nosec G304 -- root and Makefile path are repository-owned.
	if err != nil {
		return fmt.Errorf("read Makefile release target: %w", err)
	}
	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	want := "\nverify-release:\n\tgo run ./internal/qualitygate -mode=verify-release\n"
	if strings.Count(normalized, want) != 1 {
		return errors.New("verify-release target must invoke the unconditional core quality gate exactly once")
	}
	return nil
}

func checkReleaseWorkflow(root string) error {
	path := filepath.Join(root, ".github", "workflows", "release.yml")
	content, err := os.ReadFile(path) // #nosec G304 -- root and workflow path are repository-owned.
	if err != nil {
		return fmt.Errorf("read release workflow: %w", err)
	}
	if strings.ReplaceAll(string(content), "\r\n", "\n") != expectedReleaseWorkflow(modulePath) {
		return fmt.Errorf(
			"release workflow must call the protected central workflow at %s for module %s with only the explicit repository signing secret",
			releaseWorkflowRevision,
			modulePath,
		)
	}
	return nil
}

func expectedReleaseWorkflow(module string) string {
	return fmt.Sprintf(`name: Release

on:
  push:
    tags:
      - "v[0-9]*.[0-9]*.[0-9]*"

permissions: {}

jobs:
  release:
    name: Centrally verify, sign, and publish
    permissions:
      contents: write
    uses: spice-framework/.github/.github/workflows/library-release.yml@%s
    with:
      module: %s
    secrets:
      SPICE_LIBRARY_RELEASE_SIGNING_KEY: ${{ secrets.SPICE_LIBRARY_RELEASE_SIGNING_KEY }}
`, releaseWorkflowRevision, module)
}

func checkReleasePublicKey(root string) error {
	path := filepath.Join(root, "security", "release", "ed25519-public.pem")
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect release public key: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("release public key must be a regular file, not a symlink")
	}
	content, err := os.ReadFile(path) // #nosec G304 -- root and public-key path are repository-owned.
	if err != nil {
		return fmt.Errorf("read release public key: %w", err)
	}
	block, trailing := pem.Decode(content)
	if block == nil || block.Type != "PUBLIC KEY" || len(strings.TrimSpace(string(trailing))) != 0 {
		return errors.New("release public key must contain exactly one canonical PUBLIC KEY PEM block")
	}
	if !bytes.Equal(content, pem.EncodeToMemory(block)) {
		return errors.New("release public key must use canonical PEM encoding")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse release public key: %w", err)
	}
	key, ok := parsed.(ed25519.PublicKey)
	if !ok || len(key) != ed25519.PublicKeySize {
		return errors.New("release public key must be an Ed25519 SubjectPublicKeyInfo key")
	}
	digest := sha256.Sum256(block.Bytes)
	if fingerprint := hex.EncodeToString(digest[:]); fingerprint != releasePublicKeyFingerprint {
		return fmt.Errorf("release public-key DER fingerprint is %s, want %s", fingerprint, releasePublicKeyFingerprint)
	}
	return nil
}
