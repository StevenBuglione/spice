.PHONY: fmt test vet verify smoke

fmt:
	gofmt -w $$(find . -type f -name '*.go' -not -path './vendor/*' -not -path './.git/*')

test:
	go test ./...

vet:
	go vet ./...

smoke:
	go run ./cmd/spice version
	go run ./cmd/spice verify ./...
	go run ./examples/hello-world -check

verify:
	./scripts/verify.sh
