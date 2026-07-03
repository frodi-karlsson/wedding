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

`terraform.tfvars` is gitignored. Never commit it.

## Local staging

```sh
cd infra
cp .env.staging.example .env.staging
docker compose -f docker-compose.dev.yml up --build
# Backend at http://localhost:8080
```

The staging backend uses a bind-mounted `./data` directory for its
SQLite file. The distroless image runs as the `nonroot` user (uid 65532),
so give that uid ownership rather than making the directory world-writable:

```sh
mkdir -p infra/data
sudo chown 65532:65532 infra/data && chmod 750 infra/data
```

(On prod the volume at `/mnt/data` is set up the same way by cloud-init.)

Optionally seed test data:
```sh
sqlite3 infra/data/wedding-staging.db < db/seed.staging.sql
```

## Backups

Nightly SQLite backup to Cloudflare R2 (`wedding-backups` bucket) via a systemd
timer at 03:00. The backup script checks the DB file exists, snapshots it via
`sqlite3 .backup`, uploads to R2 via rclone, and pings healthchecks.io on
success. If a nightly ping is missed, healthchecks.io emails you (dead-man's
switch). On explicit failure (rclone/sqlite error), a Resend email is sent.
R2 lifecycle expires backups after 30 days.

### Restore procedure

1. Install rclone locally and configure an R2 remote (same access key/secret
   as the droplet).
2. List backups: `rclone ls r2:wedding-backups/`
3. Pull the latest: `rclone copy r2:wedding-backups/wedding-backup-<TS>.db ./`
4. Verify it opens: `sqlite3 wedding-backup-<TS>.db "SELECT count(*) FROM invites;"`
5. SSH into the droplet, stop the backend: `cd /opt/wedding && docker compose stop backend`
6. **Delete the WAL + SHM files** (critical, because the DB runs in WAL mode; stale WAL/SHM left beside the restored file will replay old frames and silently corrupt or revert the restore):
   ```sh
   rm -f /mnt/data/wedding.db-wal /mnt/data/wedding.db-shm
   ```
7. Replace the DB: `cp wedding-backup-<TS>.db /mnt/data/wedding.db`
8. Restart: `docker compose up -d backend`

### Break-glass: SSH lockout

If your IP changes and SSH is blocked, use the DigitalOcean web console:
https://cloud.digitalocean.com/droplets → your droplet → Access → Launch Web
Console. From there, update `/etc/ssh/sshd_config.d/00-hardening.conf` or
update the firewall via `tofu apply` with the new IP.

## CI

See `../.github/workflows/` and `../.github/README.md`.
