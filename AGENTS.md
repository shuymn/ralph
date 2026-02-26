<!-- Do not restructure or delete sections. Update inline when behavior changes. -->

# Repository Guidelines

## Canonical Instruction File
- Keep project-specific agent/contributor guidance in this file.
- Do not add a separate `CLAUDE.md` unless explicitly requested by maintainers.

## Project Structure & Responsibilities
- `main.go` and `cmd/ralph/*` are CLI entrypoints (`init`, `run`, `run --dry-run`).
- Put business logic under `internal/*`; keep package boundaries domain-driven.
- Core packages: `config`, `runner`, `git`, `condition`, `prd`, `init`.
- Keep design records in `docs/plans/*` and architectural decisions in `docs/adr/*`.

## Build, Test, and Development Commands
- Use Task (`Taskfile.yml`) as the default interface.
- `task build`: compile `./ralph`.
- `task run`: build and run locally.
- `task schema`: regenerate `schemas/config.schema.json` from `internal/config` types.
- `task test`: run `go test -race -shuffle=on -count=10 ./...`.
- `task lint`: run project linters (`golangci-lint`, installed to `bin/`).
- `task fmt`: apply configured Go formatters.
- `task check`: run schema drift check, lint, build, and test in CI order.
- `go test -run TestName ./internal/<pkg>`: run a focused test while developing.

## Coding & Error-Handling Conventions
- Write idiomatic Go, prefer small functions, and wrap errors with context (`fmt.Errorf("...: %w", err)`).
- Keep external schema keys in snake_case YAML tags; preserve existing lint annotations when required.
- During staged config migrations, remove legacy YAML tags to keep unknown-key validation strict, and use temporary in-memory aliases with `json:"-" yaml:"-"` only to avoid cross-package breakage.
- For typed runtime/validation errors, implement both `Code() string` and `ExitCode() int`.
- Reuse existing condition/runner semantics (`success()`, `failure()`, `always()`, `changed()`, `on_fail`).
- For loop-mode extensions, explicit role overrides are authoritative; enable automatic scheduler role selection only when role is unset.

## Testing Expectations
- Add tests in the same domain package you changed (for example, `internal/runner/*_test.go`).
- Use table-driven tests where behavior branches are non-trivial.
- Keep tests parallel-safe; current test suite consistently uses `t.Parallel()`.
- Prefer real temp dirs/subprocess behavior over heavy mocks for runner and git flows.
- For scaffold/config migrations, assert both positive outputs (new files/settings exist) and negative outputs (legacy files/settings are absent).

## Commit & Pull Request Guidelines
- Follow Conventional Commits: `<type>(<scope>): <imperative summary>` (example: `feat(runner): add dry-run mode`).
- Keep commits scoped to one logical change and include tests with behavior changes.
- PRs should include purpose, linked issue/plan/ADR, validation evidence (`task check`), and config/template impact notes.

<!-- Maintenance: Recheck this file after architecture/tooling/workflow changes; keep instructions concrete and non-duplicative with global agent defaults. -->
