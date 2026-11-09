.PHONY: help check generate dev seed backend frontend
.DEFAULT_GOAL := help
SHELL := /bin/bash

DEV_ENV := infra/.env.staging
DEV_DB := infra/data/wedding-staging.db

help:
	@echo "check     run all static analysis + tests (backend + frontend)"
	@echo "generate  regenerate frontend TS types from the Go API contract (tygo)"
	@echo "dev       run the backend (:8080, local SQLite) and frontend (:4321) together"
	@echo "seed      load db/seed.staging.sql into the local dev database"

check:
	cd backend && make check
	cd frontend && pnpm run check

generate:
	cd backend && make generate

# The backend reads the staging env file but keeps its database in infra/data and
# sends a non-Secure cookie, since local dev is plain HTTP. Ctrl-C stops both.
dev: $(DEV_ENV)
	@mkdir -p $(dir $(DEV_DB))
	@trap 'kill 0' EXIT INT TERM; \
	( set -a; . ./$(DEV_ENV); set +a; \
	  cd backend && DB_PATH=../$(DEV_DB) SECURE_COOKIE=false go run ./cmd/server/ ) & \
	( cd frontend && pnpm run dev ) & \
	wait

$(DEV_ENV):
	cp infra/.env.staging.example $(DEV_ENV)

seed:
	sqlite3 $(DEV_DB) < db/seed.staging.sql
