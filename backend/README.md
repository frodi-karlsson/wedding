# Wedding Invite Backend

Go backend for the wedding invite site. Provides the RSVP API and a
password-protected admin API for managing invites.

## Requirements

- Go 1.26.4
- Docker (for container builds)

## Configuration

All config comes from environment variables:

| Var | Required | Default | Purpose |
|-----|----------|---------|---------|
| `DB_PATH` | no | `/data/wedding.db` | SQLite file path (`:memory:` for tests) |
| `ADMIN_PASSWORD` | yes | — | Admin login password |
| `SESSION_SECRET` | yes | — | HMAC secret for session cookies |
| `RESEND_API_KEY` | yes | — | Resend API key |
| `RESEND_FROM` | yes | — | From address (e.g. `rsvp@carlaochfrodi.wedding`) |
| `RESEND_TO` | yes | — | Destination email (`frodi.carla@gmail.com`) |
| `CORS_ALLOWED_ORIGINS` | yes | — | Comma-separated allowed origins |
| `PORT` | no | `8080` | Listen port |
| `SECURE_COOKIE` | no | `true` | Set `Secure` flag on admin session cookies (set `false` for local HTTP testing) |

Never commit secrets. Use a `.env` file (gitignored) for local runs.

## Development

```sh
go test ./...              # run all tests
go run ./cmd/server/       # run locally (set env vars first)
```

## API

### Public
- `GET /invites/{id}` — invite + guests
- `POST /invites/{id}/rsvp` — submit RSVP (saves + emails)

### Admin (session cookie)
- `POST /admin/login` — `{ "password": "..." }` → sets cookie
- `POST /admin/logout`
- `GET /admin/invites` — list
- `POST /admin/invites` — `{ "name", "min_plus", "max_plus" }` → creates invite + primary guest, returns shareable link id
- `GET /admin/invites/{id}`
- `PUT /admin/invites/{id}`
- `DELETE /admin/invites/{id}`

## Docker

```sh
docker build -t wedding-backend .
docker run -p 8080:8080 --env-file .env -v wedding-data:/data wedding-backend
```
