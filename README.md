# Carla & Frodi Wedding Invite Site

A wedding invitation website with an RSVP form and an admin panel for managing
invites.

## Structure

```
wedding/
├── backend/    # Go API with SQLite, RSVP, admin, email (see backend/README.md)
├── frontend/   # Astro static site with RSVP + admin pages (see frontend/README.md)
├── db/         # schema reference + staging seed
├── infra/      # OpenTofu + docker compose (see infra/README.md)
└── .github/    # CI workflows (see .github/README.md)
```

## Quick start (local)

1. **Everything at once:** `make dev` runs the backend on `:8080` (SQLite at
   `infra/data/wedding-staging.db`, env from `infra/.env.staging`, created from the
   example on first run) and the frontend on `:4321`. Ctrl-C stops both.
   `make seed` loads `db/seed.staging.sql` into that database.

2. **Separately**, if you need to: run the backend with `go run ./cmd/server/` in
   `backend/` (env vars in `backend/README.md`), or in docker via
   `docker compose -f docker-compose.dev.yml up --build` in `infra/`. Run the
   frontend with `pnpm run dev` in `frontend/`, then open
   `http://localhost:4321/?id=<inviteId>`.

3. **Login:** `http://localhost:4321/admin`. The admin password (`ADMIN_PASSWORD`) opens the invite dashboard, the translator password (`TRANSLATOR_PASSWORD`) opens the speeches. Locally (`make dev` or the staging compose) they are `staging-admin-pw` and `staging-translator-pw`.

## Deployment

- **Backend:** DigitalOcean droplet, provisioned via OpenTofu. CI builds and
  deploys on push to `main`. See `infra/README.md`.
- **Frontend:** Cloudflare Pages. CI builds and deploys on push to `main`.
- **DNS:** Cloudflare. `api.carlaochfrodi.wedding` → droplet;
  `carlaochfrodi.wedding` → Cloudflare Pages.

## Testing

```sh
make check                  # run all checks: backend tests + frontend checks
```

Or run each project separately:

```sh
cd backend  && go test ./...
cd frontend && pnpm exec vitest run
```

## Design

See `docs/superpowers/specs/` for the design spec (not committed to git).
