# Ralph Progress Log
Started: 2026-02-25

## Codebase Patterns
- During staged config migrations, keep strict YAML key validation by removing old YAML tags and expose temporary in-memory aliases with `json:"-" yaml:"-"` to avoid cross-package breakage.

## Progress Entries
## [2026-02-25] - [task-1]: Refactor config model to command profiles
- What was implemented
  - Migrated config model to `agent.run_command` + optional `review_command`/`judge_command`.
  - Split completion model into `completion.run` and `completion.review` with `review_convergence` settings and defaults.
  - Added validation for run/review strategy constraints, `review_convergence` bounds, and legacy key rejection.
  - Updated schema generator/tests and regenerated `schemas/config.schema.json` for the new config contract.
- DoD verification results
  - `go test ./internal/config/...`: PASS
  - `task schema`: PASS
  - `git diff --exit-code -- schemas/config.schema.json`: non-zero as expected in working tree because schema artifact changed for this story.
  - `task fmt`: PASS
  - `task lint`: PASS
  - `task test`: PASS
- Learnings:
  - Patterns discovered
    - Compatibility aliases let downstream runtime tests stay green while config YAML surface changes incrementally.
  - Gotchas encountered
    - Sandbox blocks git index writes, so schema drift guard remains non-zero until changes are staged/committed outside the sandbox.
---
## [2026-02-25] - [task-2]: Add `ralph review` command surface with shared config loading
- What was implemented
  - Added CLI parsing support for `ralph review [--dry-run]` in `main.go` and updated usage output.
  - Added `cmd/ralph/review.go` with `RunReview` and `RunReviewDry` entrypoints.
  - Refactored `cmd/ralph/run.go` into a shared `runCommand` flow used by run/review, with command-specific error prefixes.
  - Added review strategy guard wiring (`completion.review.strategy` must be `review_convergence`) returning exit `22` for review mode.
  - Added `main_test.go` and `cmd/ralph/review_test.go` covering review subcommand parsing, shared config path loading, and unsupported strategy rejection.
- DoD verification results
  - `go test ./... -run 'TestRunParsesReviewCommand|TestReviewUsesSharedConfigPath|TestReviewRejectsUnsupportedStrategy'`: PASS (with local Go cache env in sandbox)
  - `task fmt`: PASS
  - `task lint`: PASS (with local cache env)
  - `task test`: PASS (with local Go cache env)
- Learnings:
  - Patterns discovered
    - Keep run/review command entrypoints on a single loader/executor path and vary behavior by explicit mode labels for consistent config-path and error handling.
  - Gotchas encountered
    - Sandbox denies writes to default Go/golangci cache locations; quality gates require repo-local cache env overrides (`GOCACHE`, `GOMODCACHE`, `GOPATH`, `GOLANGCI_LINT_CACHE`).
---
