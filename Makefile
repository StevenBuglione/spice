.PHONY: fmt fuzz lint offline security smoke test vet verify

fmt:
	go run ./internal/qualitygate -mode=fmt

fuzz:
	go run ./internal/qualitygate -mode=fuzz

lint:
	go run ./internal/qualitygate -mode=lint

offline:
	go run ./internal/qualitygate -mode=offline

security:
	go run ./internal/qualitygate -mode=security

smoke:
	go run ./internal/qualitygate -mode=smoke

test:
	go run ./internal/qualitygate -mode=test

vet:
	go run ./internal/qualitygate -mode=vet

verify:
	go run ./internal/qualitygate -mode=verify
