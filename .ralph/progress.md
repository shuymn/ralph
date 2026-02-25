# Ralph Progress Log
Started: 2026-02-25

## Codebase Patterns

## Progress Entries

## [2026-02-25] - [task-1]: Scope Freeze and Design Record Sync
- What was implemented
  - Added remediation scope freeze language to the design doc and ADR, explicitly marking run_id second-granularity as accepted risk and init quality-gate insertion as no-action.
  - Synced remediation plan header vocabulary to use the status taxonomy `fix` / `accepted` / `no-action` as canonical terms.
- DoD verification results
  - PASS: `rg -n "accepted risk|no-action|remediation scope|Findings Baseline" docs/plans/2026-02-25-review-convergence-design.md docs/adr/0004-review-convergence-mode.md docs/plans/2026-02-25-review-feedback-remediation-plan.md`
  - PASS: `task fmt`
  - PASS: `task lint`
  - PASS: `task test`
  - PASS: `task build`
- Learnings:
  - Patterns discovered
    - Scope-freeze tasks in this repo should anchor accepted/no-action boundaries in both design and ADR so later implementation tasks can reference one vocabulary.
  - Gotchas encountered
    - The planned RED grep initially passed because markers already existed in the remediation plan file; a minimal target correction (design+ADR only) was required to establish executable RED.
---
## [2026-02-25] - [task-2]: Fix Review Completion Routing for Explicit Roles
- What was implemented
  - Switched review-runtime activation to be mode-based (`ModeReview`) while keeping role scheduling gated to "role unset", so explicit `RoleReview`/`RoleJudge` still use review completion semantics.
  - Added judge fail-fast handling: non-zero judge command exits now return runtime error and are never counted toward convergence.
  - Added RED/GREEN regression coverage for explicit-role review completion routing and judge non-zero convergence behavior.
  - Updated the existing role-command fallback test to validate command fallback behavior without relying on run-mode completion semantics in review mode.
- DoD verification results
  - PASS: `go test ./internal/runner -run 'TestReviewExplicitRoleUsesReviewCompletion|TestReviewJudgeNonZeroDoesNotConverge'`
  - PASS: `task fmt`
  - PASS: `task lint`
  - PASS: `task test`
  - PASS: `task build`
- Learnings:
  - Patterns discovered
    - In review mode, completion routing should be keyed by mode contract while scheduler activation stays keyed by explicit-role presence.
  - Gotchas encountered
    - Existing tests that asserted exit `0` for explicit review roles were implicitly coupled to run-mode completion and needed role-mode-appropriate assertions.
---
## [2026-02-25] - [task-3]: Fix Partial Defaulting in `review_convergence`
- What was implemented
  - Added RED regression tests for partial `completion.review.review_convergence` overrides and single-field overrides to ensure omitted sibling fields are defaulted.
  - Switched `applyDefaults` in `internal/config/validate.go` to field-by-field default filling for `min_reviews`, `max_reviews`, `judge_every`, and `stable_rounds`.
  - Refactored duplicated assertions in new tests into `assertReviewConvergenceValues` to satisfy lint duplication checks.
  - Updated invalid-bound tests to use negative values so lower-bound validation remains covered under the partial-defaulting model.
- DoD verification results
  - PASS: `go test ./internal/config -run 'TestLoadBytesAppliesReviewConvergencePartialDefaults|TestLoadBytesReviewConvergenceSingleFieldOverrides'`
  - PASS: `task fmt`
  - PASS: `task lint`
  - PASS: `task test`
  - PASS: `task build`
- Learnings:
  - Patterns discovered
    - For optional numeric sub-objects decoded into value fields, omission-safe defaults should be applied per field rather than only on full-struct zero checks.
  - Gotchas encountered
    - Legacy tests that asserted explicit `0` rejection conflicted with omission defaulting semantics in non-pointer numeric fields and needed boundary assertions anchored on negative values.
---
## [2026-02-25] - [task-4]: Enforce Strict Judge JSON Contract
- What was implemented
  - Added regression tests `TestJudgeContractRejectsUnknownFields` and `TestJudgeContractRejectsNegativeNewFindings`.
  - Added a shared helper in judge contract tests to run review mode against injected judge artifacts with consistent convergence settings.
  - Switched judge artifact parsing to a strict JSON decoder with `DisallowUnknownFields()` while preserving required-field and negative-count validation.
- DoD verification results
  - RED (expected fail): `GOCACHE=$(pwd)/.cache/go-build go test ./internal/runner -run 'TestJudgeContractRejectsUnknownFields|TestJudgeContractRejectsNegativeNewFindings'` -> FAIL (`expected runtime exit code 22, got 0`).
  - PASS: `GOCACHE=$(pwd)/.cache/go-build go test ./internal/runner -run 'TestJudgeContractRejectsUnknownFields|TestJudgeContractRejectsNegativeNewFindings'`
  - PASS: `GOCACHE=$(pwd)/.cache/go-build task fmt`
  - PASS: `GOCACHE=$(pwd)/.cache/go-build task lint`
  - PASS: `GOCACHE=$(pwd)/.cache/go-build task test`
  - PASS: `GOCACHE=$(pwd)/.cache/go-build task build`
- Learnings:
  - Patterns discovered
    - Contract JSON that controls loop convergence should use unknown-field rejection to preserve fail-closed behavior.
  - Gotchas encountered
    - Go commands needed a local `GOCACHE` path in this sandbox to avoid permission errors on the default cache location.
---
