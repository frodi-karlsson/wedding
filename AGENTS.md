# Conventions

## Frontend testing (Vitest)

- **Framework:** Vitest 4.x. Import from `vitest`.
- **Package manager:** pnpm only (never npm/npx).
- **Environment:** happy-dom. `isolate: false` in vitest.config.ts.
- **File naming:** `*.test.ts`, co-located in `tests/`.
- **Titles:** Start with "should" — `test('should ...', async () => {})`.
- **`test` over `it`:** Use `test()`, not `it()`. Enforced by eslint-plugin-vitest.
- **AAA structure:** Arrange / Act / Assert order, separated by blank lines. No `// Arrange` comments — the blank lines and logical grouping are sufficient.
- **Assertions:** `toBe` for primitives, `toEqual` for objects/arrays. Prefer specific over generic.
- **No named spies:** `vi.spyOn(obj, 'method')` then assert on `obj.method` directly — do NOT assign to a `const spy`.
- **Mocking:** `vi.stubGlobal` for globals (e.g. fetch). `vi.spyOn` for object methods. Prefer `.mockResolvedValueOnce` over `.mockResolvedValue` to avoid leaking state.
- **Coverage:** Cover the happy path + error/edge cases. Each branch needs a test. Verify observable outcomes, not internal wiring.

## Frontend linting (ESLint)

- Flat config (`eslint.config.js`). ESLint 10.
- `no-restricted-globals`: `window` is banned — use `globalThis` instead.
- `eslint-plugin-vitest` for test conventions (`test` over `it`).
- `eslint-plugin-astro` + `astro-eslint-parser` for `.astro` files.
- No lint silencing (`// eslint-disable`) — fix the code. If a rule is genuinely anti-idiomatic, document the exception in AGENTS.md.

## Backend (Go)

- golangci-lint v2 (strict config at `backend/.golangci.yml`). No `//nolint` — fix the code.
- govulncheck for vulnerability scanning.
- TDD: failing test first, implement, pass, commit.

## General

- TDD throughout (test-first).
- No secrets committed or in plain text.
- Versions pinned to verified-latest (see spec).
