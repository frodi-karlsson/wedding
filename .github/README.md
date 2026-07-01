# GitHub Actions — CI/CD

## Workflows

- `backend.yml` — tests, builds, pushes backend image to DO Container Registry,
  SSH-deploys to the droplet. Runs on pushes/PRs touching `backend/`.
- `frontend.yml` — tests, builds, deploys frontend to Cloudflare Pages via
  wrangler. Runs on pushes/PRs touching `frontend/`.
- `infra-check.yml` — `tofu fmt -check` + `tofu validate`. Runs on PRs and pushes
  to `main` touching `infra/`.
- `lint.yml` — runs golangci-lint, govulncheck, and `go mod tidy` check on the
  backend. Runs on pushes/PRs to `main`.

## Required GitHub Secrets

| Secret | Used by | Purpose |
|--------|---------|---------|
| `DO_TOKEN` | backend, infra | DigitalOcean API token |
| `DO_REGISTRY_ENDPOINT` | backend | DO Container Registry endpoint, e.g. `registry.digitalocean.com/wedding` — must include the registry name (the `registry_endpoint` OpenTofu output) |
| `DROPLET_IP` | backend | Droplet public/reserved IP for SSH deploy |
| `DROPLET_SSH_KEY` | backend | SSH private key for droplet access |
| `CLOUDFLARE_API_TOKEN` | frontend | Cloudflare API token (Pages deploy) |
| `CLOUDFLARE_ACCOUNT_ID` | frontend | Cloudflare account ID |

The backend deploys only on pushes to `main` (not PRs). PRs run tests + build only.

## Local OpenTofu

`tofu plan` / `tofu apply` run locally (by you) using secrets in
`infra/prod/terraform.tfvars`. CI does NOT run `plan` or `apply`.
