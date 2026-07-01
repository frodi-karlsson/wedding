# Infrastructure

OpenTofu-managed infrastructure for the wedding invite site, plus docker
compose files for prod and local staging.

## Resources provisioned

**DigitalOcean:**
- Droplet (runs Caddy + backend via docker compose)
- Volume (1GB, mounted at `/mnt/data` for the SQLite file)
- Firewall (ports 22, 80, 443)
- Reserved IP (stable across droplet recreation)
- Container Registry (private, for the backend image)

**Cloudflare:**
- Pages project (direct-upload, no git connection)
- Pages custom domain (apex → carlaochfrodi.wedding)
- DNS: `api.` A record → droplet reserved IP (DNS-only)
- DNS: apex CNAME → `<project>.pages.dev`

## Prerequisites

1. Install OpenTofu 1.12.3: https://opentofu.org/docs/intro/install/
2. A DigitalOcean account + API token (`do_token`).
3. A Cloudflare account + API token with Pages + DNS edit perms.
4. Your SSH public key for droplet access.
5. A Resend account + API key (domain `carlaochfrodi.wedding` verified in Resend).

## Setup

```sh
cd infra/prod
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your real secrets.
tofu init
tofu plan
tofu apply
```

`terraform.tfvars` is gitignored — never commit it.

## Local staging

```sh
cd infra
cp .env.staging.example .env.staging
docker compose -f docker-compose.dev.yml up --build
# Backend at http://localhost:8080
```

The staging backend uses a named Docker volume (`wedding-staging-data`) for its
SQLite file, so no host permission fixes are needed.

Optionally seed test data:
```sh
sqlite3 infra/data/wedding-staging.db < db/seed.staging.sql
```

## Backups

Backups are manual. The SQLite file lives on a DO Volume (durable, persists
across droplet recreation). To take a snapshot after a batch of RSVPs, SSH into
the droplet and run:

```sh
sqlite3 /mnt/data/wedding.db ".backup '/tmp/wedding-backup.db'"
# then scp /tmp/wedding-backup.db down to your machine
```

There is no automated off-droplet backup (DO Spaces was removed to avoid the
$5/mo flat cost — not worth it for a few KB of RSVP data).

## CI

See `../.github/workflows/` and `../.github/README.md`.
