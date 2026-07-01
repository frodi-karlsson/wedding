# ESLint + Vitest conventions report

## Part 1: ESLint setup (flat config, ban `window`)

### Installed dependencies
- `eslint@10.6.0`
- `eslint-plugin-astro@2.1.1`
- `typescript-eslint@8.62.1`

Note: the requested `@astrojs/eslint-parser` package does not exist on npm. `eslint-plugin-astro` already depends on `astro-eslint-parser`, so the Astro parser is available without a separate install.

### New config
`frontend/eslint.config.js` (flat config):
- `typescript-eslint` `recommended` for `**/*.ts`.
- `eslint-plugin-astro` `recommended` for `**/*.astro`.
- `no-restricted-globals` with `'window'` and the message "Use `globalThis` instead of `window` for portability." (`error`).
- `no-restricted-syntax` with `MemberExpression[object.name="window"]` as a fallback (`error`).
- Ignores `dist/`, `node_modules/`, `.astro/`.
- `src/env.d.ts` override: disables `@typescript-eslint/triple-slash-reference`, because this is the Astro-generated client-types reference file and must keep triple-slash references.

### New script
`frontend/package.json`:
```json
"lint": "eslint ."
```

### `window` → `globalThis` changes
| File | Line | Change |
|------|------|--------|
| `frontend/src/scripts/admin.ts` | 19 | `window.location.origin` → `globalThis.location.origin` |
| `frontend/src/scripts/admin.ts` | 208 | `window.alert(...)` → `globalThis.alert(...)` |
| `frontend/src/scripts/admin.ts` | 253 | `window.confirm(...)` → `globalThis.confirm(...)` |

### Other lint fixes
- `frontend/tests/api.test.ts`: removed unused `input`/`init` parameters from the `vi.fn` stub in `setupFetch` (now uses a no-arg arrow function).

### Verification
```text
$ pnpm exec eslint .
(no output)
```

```text
$ grep -R "window\." frontend/src/ || true
No window. usage in src/
```

## Part 2: Vitest isolate disabled

`frontend/vitest.config.ts` updated:
```ts
export default getViteConfig({
  test: { environment: 'jsdom', globals: true, isolate: false },
});
```

## Part 3: Testing conventions applied

Files updated:
- `frontend/tests/i18n.test.ts`
- `frontend/tests/api.test.ts`
- `frontend/tests/rsvp-form.test.ts`
- `frontend/tests/admin.test.ts`

Conventions applied per file:

### `i18n.test.ts`
- Renamed all `it(...)` titles to start with `should`.
- Added blank-line AAA grouping (arrange / act / assert).

### `api.test.ts`
- Titles already started with `should`; no renames needed.
- Added blank-line AAA grouping between setup, the `await api.*().run()` call, and the assertion block.
- Removed unused `input`/`init` parameters from the mock `fetch` function to satisfy lint.

### `rsvp-form.test.ts`
- Titles already started with `should`; no renames needed.
- Added blank-line AAA grouping between setup, action, and assertions.
- Kept `toEqual` for object assertions; `toBe` for primitives.

### `admin.test.ts`
- Titles already started with `should`; no renames needed.
- Replaced `vi.stubGlobal('window', ...)` with `vi.stubGlobal('location', { origin: 'https://example.com' })` to match the new `globalThis.location.origin` usage in `buildShareLink`.
- Added blank-line AAA grouping.

## Verification output

### ESLint
```text
$ pnpm exec eslint .
(no output)
```

### Vitest
```text
$ pnpm exec vitest run

 RUN  v4.1.9 /Users/frodi/Development/wedding/frontend

 Test Files  4 passed (4)
      Tests  32 passed (32)
   Start at  23:56:14
   Duration  615ms
```

### Build
```text
$ pnpm run build
...
23:56:22 [build] 10 page(s) built in 307ms
23:56:22 [build] Complete!
```

## Concerns / rule exceptions
- `@astrojs/eslint-parser` was unavailable; used the `astro-eslint-parser` dependency that ships with `eslint-plugin-astro`.
- `src/env.d.ts` is exempt from `@typescript-eslint/triple-slash-reference` because it is the Astro-generated reference file that must use `/// <reference ... />` directives for client types.
