.PHONY: benchmark check coverage fmt fuzz goland lint offline security smoke test vet verify verify-release zed

benchmark:
	go run ./internal/qualitygate -mode=benchmark

check:
	go run ./internal/qualitygate -mode=check

coverage:
	go run ./internal/qualitygate -mode=coverage

fmt:
	go run ./internal/qualitygate -mode=fmt

fuzz:
	go run ./internal/qualitygate -mode=fuzz

goland:
	go run ./internal/qualitygate -mode=goland

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

zed:
	go run ./internal/qualitygate -mode=zed

verify:
	go run ./internal/qualitygate -mode=verify

verify-release:
	go run ./internal/qualitygate -mode=verify-release
