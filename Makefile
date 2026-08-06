.PHONY: check coverage fast fmt fuzz lint offline security test vet verify verify-release

check:
	go run ./internal/qualitygate -mode=check

coverage:
	go run ./internal/qualitygate -mode=coverage

fast:
	go run ./internal/qualitygate -mode=fast

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

test:
	go run ./internal/qualitygate -mode=test

vet:
	go run ./internal/qualitygate -mode=vet

verify:
	go run ./internal/qualitygate -mode=verify

verify-release:
	go run ./internal/qualitygate -mode=verify-release
