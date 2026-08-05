.PHONY: benchmark bootstrap check coverage dogfood fast fmt fuzz lint offline security smoke test vet verify verify-release

benchmark:
	go run ./internal/qualitygate -mode=benchmark

bootstrap:
	go run ./internal/qualitygate -mode=bootstrap

check:
	go run ./internal/qualitygate -mode=check

coverage:
	go run ./internal/qualitygate -mode=coverage

dogfood:
	go run ./internal/qualitygate -mode=dogfood

fast:
	go run ./internal/qualitygate/fast

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

verify-release:
	go run ./internal/qualitygate -mode=verify-release
