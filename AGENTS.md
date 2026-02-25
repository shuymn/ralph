<!-- Do not restructure or delete sections. Update inline when workflow changes. -->

## Core Rules
- Execute only requested scope; do not add unrequested features.
- Do not preserve backward compatibility unless explicitly requested.
- Use constants or configuration; do not hardcode environment-specific values.
- Fix root causes; do not silence lint or test failures.
- If requirements are unclear, ask one clarifying question before editing.

## Development Workflow
- Use `task` as the standard interface for local work.
- Run `task check` before committing or opening a PR.
- Use `task fmt` and `task lint` for Go formatting and lint compliance.
- Use `task test` for normal validation and `task test:coverage` when coverage evidence is required.
- Run `task deps-verify` when `go.mod` or tooling versions change.

## Project Conventions
- Keep architecture decisions in `docs/adr/`.
- Keep design and plan artifacts in `docs/plans/`.
- Treat `.ralph/` as workflow runtime state; edit only when workflow behavior changes.
- Add code and tests together (`<name>.go` and `<name>_test.go`).

## Commit and PR Expectations
- Use conventional commit prefixes (`feat`, `fix`, `docs(...)`, `chore`).
- Keep each commit focused on one logical concern.
- PRs must include scope, linked issue/task (if any), and validation results (`task check`).
- Update docs when behavior, architecture, or workflow changes.
- Ensure pre-commit hooks pass for staged Go files.

<!-- Maintenance: Re-audit when workflow or tooling changes, or when this file exceeds ~30 instruction lines. -->
