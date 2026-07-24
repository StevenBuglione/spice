#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

echo "==> checking formatting"
mapfile -d '' go_files < <(find . -type f -name '*.go' -not -path './vendor/*' -not -path './.git/*' -print0)
if ((${#go_files[@]} > 0)); then
  unformatted="$(gofmt -l "${go_files[@]}")"
  if [[ -n "${unformatted}" ]]; then
    echo "These files are not gofmt-formatted:" >&2
    echo "${unformatted}" >&2
    exit 1
  fi
fi

echo "==> running go vet"
go vet ./...

echo "==> running tests"
go test ./...

echo "==> running Spice source verification"
go run ./cmd/spice verify ./...

echo "==> running CLI smoke test"
go run ./cmd/spice version

echo "==> running example smoke test"
go run ./examples/hello-world -check

echo "==> all verification passed"
