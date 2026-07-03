# Wedding Invite Frontend

Astro static site for the wedding invite RSVP + admin pages. Deployed to
Cloudflare Pages. Localized in English, Icelandic, German, and Swedish.

## Requirements

- Node.js 22+
- pnpm (never npm/npx)

## Development

```sh
pnpm install
cp .env.example .env    # set PUBLIC_API_URL to your local backend
pnpm run dev            # http://localhost:4321
```

Start the backend first (see `../backend/README.md`).

## Testing

```sh
pnpm exec vitest run        # all tests
pnpm exec vitest run tests/i18n.test.ts --silent=false   # single file
```

## Linting

```sh
pnpm exec eslint .
```

## Build

```sh
pnpm run build           # outputs dist/
```

## Environment

| Var | Purpose |
|-----|---------|
| `PUBLIC_API_URL` | Backend base URL (e.g. `http://localhost:8080` dev, `https://api.carlaochfrodi.wedding` prod) |

Set in `.env` for local dev, and as a Cloudflare Pages build-time env var for prod.

## Internationalization (i18n)

Four languages: English (`en`), Icelandic (`is`), German (`de`), Swedish (`sv`).

- **Routing:** Astro i18n routing (`astro.config.mjs`). English is at `/` (no
  prefix); the others at `/is`, `/de`, `/sv`. Pages live under
  `src/pages/[locale]/` with a root `src/pages/` copy for the English routes
  (since `prefixDefaultLocale: false` doesn't generate a root page from a
  `[locale]` dynamic route).
- **Translations:** plain JSON at `src/locales/{en,is,de,sv}.json`. A single
  `translate(key, lang)` utility (`src/scripts/i18n.ts`) imports all four and
  falls back to English, then the key string. It runs both at build time (in
  `.astro` frontmatter) and at runtime (in client islands).
- **Language picker:** `src/components/LangPicker.astro` navigates between
  locale paths, preserving the current path and query string.
- **Adding/updating translations:** edit `src/locales/<lang>.json`. The key set
  must be identical across all four files (a key-completeness test is planned).

## Pages

- `/` (and `/<lang>`) is the RSVP form. Pass `?id=<inviteId>` to load a specific
  invitation. The form prefills the invite's preset guests and lets the invitee
  add/remove pluses up to `max_plus`, fill names/dietary/alcohol-free, and
  submit.
- `/admin` (and `/<lang>/admin`) is the admin panel (password-protected via the
  backend). Manage invites: create with preset guest names + a link-language
  picker, edit, delete, copy shareable links.

## Conventions

See `../AGENTS.md` for testing, linting, and coding conventions.

## Deployment

Deployed to Cloudflare Pages via `pnpm exec wrangler pages deploy dist` in CI.
See `../infra/` and the CI workflow for details.
