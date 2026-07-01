.PHONY: help check backend frontend
.DEFAULT_GOAL := help

help:
	@echo "check    run all static analysis + tests (backend + frontend)"

check:
	cd backend && make check
	cd frontend && pnpm run check
