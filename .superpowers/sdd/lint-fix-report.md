# Lint Fix Report

Fixed 23 `golangci-lint` findings across the Go backend. All changes are in the `backend/` directory.

## err113 — use `errors.Is` + define sentinel errors

1. `backend/cmd/server/main.go:49` — changed `err != http.ErrServerClosed` to `!errors.Is(err, http.ErrServerClosed)` and added the `errors` import.
2. `backend/internal/config/config.go:60` — defined `var ErrMissingEnvVars = errors.New("missing required env vars")` and wrapped the error: `fmt.Errorf("%w: %s", ErrMissingEnvVars, strings.Join(missing, ", "))`.
3. `backend/internal/db/db.go:78` — defined exported `var ErrNoGuestNames = errors.New("at least one guest name is required")` and returned it from `CreateInvite`.
4. `backend/internal/db/db.go:117` — changed `err == sql.ErrNoRows` to `errors.Is(err, sql.ErrNoRows)`.
5. `backend/internal/db/db.go:177` — reused `ErrNoGuestNames` in `UpdateInvite`.
6. `backend/internal/db/migrate.go:40` — defined `var ErrParseMigrationVersion = errors.New("parse migration version")` and returned `fmt.Errorf("%w: no underscore separator in %s", ErrParseMigrationVersion, name)`.

## errcheck — unchecked `tx.Rollback`

7. `backend/internal/db/db.go:84` — `defer tx.Rollback()` → `defer func() { _ = tx.Rollback() }()` in `CreateInvite`.
8. `backend/internal/db/db.go:183` — same fix in `UpdateInvite`.
9. `backend/internal/db/db.go:270` — same fix in `SubmitRSVP`.

## gocritic — `exitAfterDefer` + `hugeParam`

10. `backend/cmd/server/main.go` — removed `defer d.Close()`; `d.Close()` is now called explicitly before every `log.Fatalf` that runs while the DB is open, and at the end of `main` after server shutdown. This resolves the `exitAfterDefer` findings for open-db, migrate, and server errors.
11. `backend/internal/invite/invite.go:90` — `func validate(inv db.Invite, ...)` → `func validate(inv *db.Invite, ...)`; call site changed to `validate(&inv, guests)`.
12. `backend/internal/invite/invite.go:115` — `func buildRSVPEmailBody(inv db.Invite, ...)` → `func buildRSVPEmailBody(inv *db.Invite, ...)`; call site changed to `buildRSVPEmailBody(&inv, guests)`.
13. `backend/internal/server/types.go:64` — `func toGuestResponse(g db.Guest)` → `func toGuestResponse(g *db.Guest)`; range call sites now pass `&guests[i]`.
14. `backend/internal/server/types.go:74` — `func toInviteResponse(inv db.Invite)` → `func toInviteResponse(inv *db.Invite)`; all call sites changed to `toInviteResponse(&inv)`.

## modernize — `interface{}` → `any`

15. `backend/internal/server/server.go:43` — `func writeJSON(w ..., v interface{})` → `v any`.
16. `backend/internal/server/server.go:55` — `func decodeJSON(r ..., v interface{})` → `v any`.

## noctx — use context-aware DB methods

17. `backend/internal/db/db.go:23` — `d.Exec("PRAGMA ...")` → `d.ExecContext(context.Background(), "PRAGMA ...")`.
18. `backend/internal/db/migrate.go:17` — `d.Exec(...)` → `d.ExecContext(context.Background(), ...)`.
19. `backend/internal/db/migrate.go:49` — `d.QueryRow(...)` → `d.QueryRowContext(context.Background(), ...)`.
20. `backend/internal/db/migrate.go:61` — `d.Exec(string(content))` → `d.ExecContext(context.Background(), string(content))`.
21. `backend/internal/db/migrate.go:64` — `d.Exec(...)` → `d.ExecContext(context.Background(), ...)`.

## rowserrcheck + sqlclosecheck

22. `backend/internal/db/db.go:195` — after the `rows.Next()` loop in `UpdateInvite`, added `if err := rows.Err(); err != nil { return Invite{}, err }`.
23. `backend/internal/db/db.go:204` — moved `rows.Close()` to `defer rows.Close()` immediately after the `QueryContext` succeeds and before the loop.

## Verification

Commands run from `backend/`:

```bash
go test ./...
```

```
?   	wedding/backend/cmd/server	[no test files]
ok  	wedding/backend/internal/auth	0.345s
ok  	wedding/backend/internal/config	0.143s
ok  	wedding/backend/internal/db	0.652s
ok  	wedding/backend/internal/email	0.772s
ok  	wedding/backend/internal/invite	0.928s
ok  	wedding/backend/internal/server	1.079s
```

```bash
go vet ./...
```

```
(no output)
```

```bash
golangci-lint run
```

```
0 issues.
```

All 23 findings are resolved; `golangci-lint`, `go test`, and `go vet` are clean.
