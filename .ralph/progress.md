# Ralph Progress Log
Started: 2026-02-25

## Codebase Patterns
- During staged config migrations, keep strict YAML key validation by removing old YAML tags and expose temporary in-memory aliases with `json:"-" yaml:"-"` to avoid cross-package breakage.
- For mode-based loop extensions, keep explicit role overrides authoritative and enable automatic scheduler behavior only when role is unset.

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
## [2026-02-25] - [task-3]: Introduce mode/role command and prompt resolution in runner
- What was implemented
  - Added mode/role-aware runner planning (`ModeRun`/`ModeReview`, `RoleRun`/`RoleReview`/`RoleJudge`) with prompt path selection for `.ralph/prompt.run.md`, `.ralph/prompt.review.md`, and `.ralph/prompt.judge.md`.
  - Wired main-step execution through a role-aware plan object so prompt and command resolution are explicit at runtime.
  - Added `internal/runner/mode_plan.go` for command fallback (`review_command`/`judge_command` -> `run_command`) plus prompt existence/non-empty validation.
  - Added Task 3 runner tests for run prompt path usage, review prompt requirements, role fallback behavior, and run tail-match compatibility; refactored the prompt-missing cases into table-driven form.
  - Resolved lint/root-cause issues in Task 3 files (static sentinel errors, exhaustive switch handling, and intentional shell execution annotation).
- DoD verification results
  - `go test ./internal/runner -run 'TestRunUsesPromptRunPath|TestReviewRequiresPromptFiles|TestRoleCommandFallback|TestRunTailMatchCompatibility'`: PASS
  - `task fmt`: PASS
  - `task lint`: PASS
  - `task test`: PASS
- Learnings:
  - Patterns discovered
    - Centralizing mode/role resolution into a dedicated plan object keeps loop orchestration stable while enabling role-specific command/prompt contracts.
  - Gotchas encountered
    - Repository-local Go module caches can contain read-only files; transient cache cleanup may require permission normalization before deletion in sandboxed runs.
---
## [2026-02-25] - [task-4]: Implement review scheduler state and artifact lifecycle
- What was implemented
  - Added review runtime state (`review_count`, `reviews_since_judge`, `judge_count`, `run_id`) with UTC `YYYYMMDDTHHMMSSZ` formatting.
  - Added pure review role scheduler logic for `min_reviews`, `judge_every`, and `max_reviews` boundaries.
  - Added deterministic artifact path management for `.ralph/reviews/<run_id>/REVIEW_0001.md` and `JUDGE_0001.json`.
  - Integrated review-mode loop scheduling/persistence in `internal/runner/loop.go` while preserving explicit role override behavior.
  - Added `internal/runner/review_scheduler_test.go` covering scheduling rules, max-review boundary behavior, and run_id/artifact naming.
- DoD verification results
  - `go test ./internal/runner -run 'TestReviewSchedulerRules|TestReviewSchedulerMaxReviewsBoundary|TestReviewRunIDFormatAndArtifactNaming'`: PASS
  - `task fmt`: PASS
  - `task lint`: PASS
  - `task test`: PASS
- Learnings:
  - Patterns discovered
    - Review artifact persistence can be layered onto existing temp-output execution by copying role outputs into deterministic review directories.
  - Gotchas encountered
    - Full-suite regressions can occur when auto-scheduling overrides previously explicit role pathways; keeping role override precedence avoids compatibility breaks.
---
