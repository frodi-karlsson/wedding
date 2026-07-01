.PHONY: help test lint vet vuln tidy generate check
.DEFAULT_GOAL := help

help:
	@echo "test      run all tests"
	@echo "lint      run golangci-lint"
	@echo "vet       run go vet"
	@echo "vuln      run govulncheck"
	@echo "tidy      verify go.mod/go.sum are tidy"
	@echo "generate  regenerate frontend TS types from Go (tygo)"
	@echo "check     run all static analysis + tests"

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

generate:
	tygo generate

check: lint vet vuln tidy test
