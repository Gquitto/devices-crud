# CLAUDE.md

Guidance for AI agents working in this repository.

## Workflow contract: the user writes tests, the agent writes code

This project is developed **test-first, with a strict split of ownership**:

- **The user owns all tests.** Every `*_test.go` file and everything under any `testdata/` directory is written by the user.
- **The agent must never create, edit, rename or delete test files or test fixtures**, not even to fix a typo, add a helper, or "make it compile". A `PreToolUse` hook in `.claude/settings.json` enforces this and will block such attempts.
- **The agent writes production code only** to make the user's tests pass.

### Iteration loop

1. Run `make test` (or a narrower `go test ./internal/<pkg>/... -run <Name>`).
2. Read the failures, including compile errors in test files: they define the names, signatures and behavior the production code must provide.
3. Write the **smallest** production change that makes the failing tests pass. Do not add features, endpoints or abstractions that no test asks for.
4. Re-run the tests. Repeat until green, then run `make vet`.
5. Refactor production code only while tests stay green.

### When to stop and ask

- A test looks wrong, contradicts `api/openapi.yaml` or the domain rules below, or contradicts another test.
- Passing a test would require hard-coding the expected values or otherwise cheating.
- A test needs something outside production code (new tooling, a changed fixture, a different DB).

Explain the problem and wait. Never work around a test by changing it, skipping it, or special-casing test input.

## Domain

Device: `id`, `name`, `brand`, `state` (`available` | `in-use` | `inactive`), `creationTime`.

Operations: create; full (PUT) and partial (PATCH) update; fetch one; fetch all; fetch by brand; fetch by state; delete one.

Rules:
- `creationTime` cannot be updated.
- `name` and `brand` cannot be updated while the device is `in-use`.
- `in-use` devices cannot be deleted.
- An empty `state` on create defaults to `available`.
- `name` and `brand` must not be null or blank (whitespace-only counts as blank), on create and on update.
- The in-use rule is evaluated against the device's **current** state, before the update is applied. Changing only `state`, or sending unchanged `name`/`brand` values, is allowed. Freeing a device and renaming it in the same request is rejected.
- A failed update leaves the device unchanged.

The HTTP contract draft lives in [api/openapi.yaml](api/openapi.yaml). Tests take precedence over it. When production code changes the contract to satisfy tests, update the YAML too.

## Architecture

```
cmd/api/                  entrypoint: config from env, wiring, HTTP server
internal/device/          domain: entity, state, rules, errors, service, repository interface
internal/storage/sqlite/  repository implementation on SQLite (modernc.org/sqlite, no CGO)
  migrations/             SQL migrations, embedded with go:embed
internal/transport/http/  gin handlers, routing, DTOs, domain error → HTTP status mapping
api/openapi.yaml          API documentation
```

Dependency direction: `transport/http` → `device` ← `storage/sqlite`. The `device` package must not import gin, `database/sql` or any driver.

## Conventions

- **English only** for code, comments, docs, commit messages and error strings.
- Go 1.27, stdlib first; gin for HTTP; `modernc.org/sqlite` for persistence. Add a dependency only when a test needs it.
- Format with `gofmt`; keep `go vet ./...` clean.
- Config through env vars: `PORT` (default `8080`), `DB_PATH` (SQLite file path).

## Commands

| Command | Purpose |
|---|---|
| `make test` | All tests with `-race` |
| `make cover` | Tests with coverage summary |
| `make vet` / `make fmt` | Static checks / formatting |
| `make run` | Run the API locally |
| `make up` / `make down` | Start / stop with Docker Compose |
