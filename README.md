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

1. **Backend:**
   ```sh
   cd backend
   # set env vars (see backend/README.md)
   go run ./cmd/server/
   ```
   Or via staging compose:
   ```sh
   cd infra
   cp .env.staging.example .env.staging
   docker compose -f docker-compose.dev.yml up --build
   ```

2. **Frontend:**
   ```sh
   cd frontend
   pnpm install
   cp .env.example .env   # PUBLIC_API_URL=http://localhost:8080
   pnpm run dev
   ```
   Open `http://localhost:4321/?id=<inviteId>`.

3. **Login:** `http://localhost:4321/admin`. The admin password (`ADMIN_PASSWORD`) opens the invite dashboard, the translator password (`TRANSLATOR_PASSWORD`) opens the speeches. With the staging compose these are `staging-admin-pw` and `staging-translator-pw`.

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
