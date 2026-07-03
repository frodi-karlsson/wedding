# Conventions

## Frontend testing (Vitest)

- **Framework:** Vitest 4.x. Import from `vitest`.
- **Package manager:** pnpm only (never npm/npx).
- **Environment:** happy-dom. `isolate: false` in vitest.config.ts.
- **File naming:** `*.test.ts`, co-located in `tests/`.
- **Titles:** Start with "should", e.g. `test('should ...', async () => {})`.
- **`test` over `it`:** Use `test()`, not `it()`. Enforced by eslint-plugin-vitest.
- **AAA structure:** Arrange / Act / Assert order, separated by blank lines. No `// Arrange` comments. The blank lines and logical grouping are sufficient.
- **Assertions:** `toBe` for primitives, `toEqual` for objects/arrays. Prefer specific over generic.
- **No named spies:** `vi.spyOn(obj, 'method')` then assert on `obj.method` directly. Do NOT assign to a `const spy`.
- **Mocking:** `vi.stubGlobal` for globals (e.g. fetch). `vi.spyOn` for object methods. Prefer `.mockResolvedValueOnce` over `.mockResolvedValue` to avoid leaking state.
- **Coverage:** Cover the happy path + error/edge cases. Each branch needs a test. Verify observable outcomes, not internal wiring.

## Frontend linting (ESLint)

- Flat config (`eslint.config.js`). ESLint 10.
- `no-restricted-globals`: `window` is banned. Use `globalThis` instead.
- `eslint-plugin-vitest` for test conventions (`test` over `it`).
- `eslint-plugin-astro` + `astro-eslint-parser` for `.astro` files.
- No lint silencing (`// eslint-disable`). Fix the code. If a rule is genuinely anti-idiomatic, document the exception in AGENTS.md.

## Backend (Go)

- golangci-lint v2 (strict config at `backend/.golangci.yml`). No `//nolint`. Fix the code.
- govulncheck for vulnerability scanning.
- TDD: failing test first, implement, pass, commit.
- `make check` (in `backend/`) runs golangci-lint + go vet + govulncheck + go mod tidy check + go test.

## Comments and prose

- Prefer self-documenting names over comments. A comment that only restates what the code does is noise. Encode the intent in a name instead (e.g. rename a type to `AdminAuthenticatedView` rather than commenting why `login` is not a value). Keep comments for genuinely non-obvious rationale, and keep those terse.
- No em dashes or en dashes anywhere in code, comments, docs, or config. Use periods or commas, and reword if needed. Never substitute a semicolon. User-facing text is exempt: locale copy in `src/locales/*.json`, and rendered UI glyphs such as an em dash empty-value placeholder.

## General

- TDD throughout (test-first).
- No secrets committed or in plain text.
- Versions pinned to verified-latest (see spec).
- **`make check`** at the repo root runs all static analysis + tests for both backend and frontend. Run it before handing over work.
