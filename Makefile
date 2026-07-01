.PHONY: help check generate backend frontend
.DEFAULT_GOAL := help

help:
	@echo "check     run all static analysis + tests (backend + frontend)"
	@echo "generate  regenerate frontend TS types from the Go API contract (tygo)"

check:
	cd backend && make check
	cd frontend && pnpm run check

generate:
	cd backend && make generate
