.PHONY: help test lint vet vuln tidy check
.DEFAULT_GOAL := help

help:
	@echo "test   run all tests"
	@echo "lint   run golangci-lint"
	@echo "vet    run go vet"
	@echo "vuln   run govulncheck"
	@echo "tidy   verify go.mod/go.sum are tidy"
	@echo "check  run all static analysis + tests"

test:
	go test ./...

lint:
	golangci-lint run

vet:
	go vet ./...

vuln:
	govulncheck ./...

tidy:
	go mod tidy
	@git diff --exit-code go.mod go.sum || (echo "go.mod/go.sum are not tidy; run 'go mod tidy' and commit" && exit 1)

check: lint vet vuln tidy test
