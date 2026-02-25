# Ralph Progress Log
Started: 2026-02-25

## Codebase Patterns
- Prefer a template manifest (`output path`, `template path`, `render`) plus one shared write pipeline for consistent scaffold diagnostics.
- Classify loader failures into `parse` vs `validation` error kinds so runner exit mapping can stay deterministic.

## Progress Entries
## [2026-02-25] - [task-1]: Bootstrap `ralph init` scaffolding
- What was implemented
  - Added a Go CLI entrypoint for `ralph init` and a root `main` bootstrap.
  - Implemented `.ralph/` scaffold generation with embedded templates.
  - Added skip-if-exists behavior with stderr reporting and `progress.md` date rendering.
  - Added tests covering file creation, skip behavior, and config schema header output.
- Files changed
  - `main.go`
  - `cmd/ralph/init.go`
  - `internal/init/scaffold.go`
  - `internal/init/scaffold_test.go`
  - `internal/init/templates/config.tmpl`
  - `internal/init/templates/prompt.tmpl`
  - `internal/init/templates/prd.tmpl`
  - `internal/init/templates/progress.tmpl`
- DoD verification results
  - `task fmt`: pass
  - `task lint`: pass
  - `task test`: pass
  - `task check`: pass
- Learnings:
  - Patterns discovered
    - A single helper path for `exists -> render/copy -> write` keeps error messages and skip behavior deterministic.
  - Gotchas encountered
    - Sandbox-safe execution required setting `GOCACHE`, `GOMODCACHE`, `GOPATH`, and `GOLANGCI_LINT_CACHE` to writable paths.
---
## [2026-02-25] - [task-2]: Add config and `prd.json` load-time validation
- What was implemented
  - Added config model/types plus load pipeline that parses YAML, applies defaults, and validates semantic constraints before runner use.
  - Implemented strict step/git validation (`name`, `run`/`uses` exclusivity, builtin restriction, `on_fail`, commit mode) with typed parse/validation errors.
  - Implemented `prd.json` load-time validation for required `stories`, required story fields, duplicate/empty IDs, and parse/type error separation.
  - Added tests for all required invalid fixtures and defaulting behavior.
- Files changed
  - `go.mod`
  - `go.sum`
  - `internal/config/types.go`
  - `internal/config/load.go`
  - `internal/config/validate.go`
  - `internal/config/load_test.go`
  - `internal/prd/validate.go`
  - `internal/prd/validate_test.go`
- DoD verification results
  - `task fmt`: pass
  - `task lint`: pass
  - `task test`: pass
  - `task check`: pass
  - `task deps-verify`: pass
- Learnings:
  - Patterns discovered
    - Generic `map[string]any` decoding works well when external schemas require snake_case keys and lint rules disallow those tag styles in Go structs.
  - Gotchas encountered
    - `go test` and `task` commands require writable cache env vars in sandbox (`GOCACHE`, `GOMODCACHE`, `GOPATH`, `GOLANGCI_LINT_CACHE`).
---
