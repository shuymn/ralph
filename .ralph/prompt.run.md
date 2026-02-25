# Ralph Agent Instructions

<!--
Editing guide:
- Keep this prompt concise and persistent; move one-off notes to `.ralph/progress.md`.
- Prefer concrete, testable rules over abstract wording.
- Update Context/Rules/Quality Gates to match the repository, but keep the one-story-per-turn workflow.
- Do not modify Task Selection, Turn Procedure, Progress Format, Codebase Patterns, or Stop Condition — these are ralph runtime machinery.
-->

## Context

You are running in the `ralph` repository root (Go CLI project).
Loop files are in `.ralph/`, plans live in `docs/plans/`, and ADRs live in `docs/adr/`.
CLI entrypoints are `main.go` and `cmd/ralph/*` (`init`, `run`, `review`, and dry-run variants).
Business logic is domain-driven under `internal/*`, with core packages `config`, `runner`, `git`, `condition`, `prd`, and `init`.
Active plan: `docs/plans/2026-02-25-review-feedback-remediation-plan.md`.
Plan focus: accepted/no-action scope freeze plus remediation of review-feedback findings via fail-fast validation and regression tests.
Architecture/stack in scope: mode-based review completion (not scheduler-presence based), strict judge contract handling, config load-time fail-closed checks, Go + YAML validation + JSON decode + schema generator + Task.
<!-- do not edit: plan resolution logic is ralph runtime machinery -->
Resolve the implementation plan file as follows:
1. Read `.ralph/prd.json`.
2. If top-level `plan` is a non-empty string, use that file.
3. Otherwise use the active plan file under `docs/plans/`.

## Rules

- Follow `AGENTS.md` and the selected task's `Files` list as the primary scope boundary.
- Implement exactly ONE story per turn with strict TDD (`RED -> GREEN -> REFACTOR`); RED must be executable test failure, never compile/import failure.
- Use the `execute-plan` skill to implement each story. When the Turn Procedure says 'Implement exactly that one story with TDD', invoke `execute-plan` with the corresponding task ID from the plan.
- Keep edits in the planned domains: `internal/runner/*`, `internal/config/*`, `internal/config/schema/*`, `schemas/config.schema.json`, `cmd/ralph/*_test.go`, `main_test.go`, `internal/init/*`, `README.md`, and docs (`docs/plans/*`, `docs/adr/*`) only when the selected task requires them.
- Do not expand scope beyond findings remediation. Accepted/no-action items are fixed boundaries: keep run_id timestamp granularity unchanged and do not force init quality-gate step insertion.
- Write idiomatic Go with small functions and wrapped errors (`fmt.Errorf("...: %w", err)`).
- Keep YAML keys snake_case and preserve required lint annotations.
- For typed runtime/validation errors, implement both `Code() string` and `ExitCode() int`.
- Reuse existing runner/condition semantics (`success()`, `failure()`, `always()`, `changed()`, `on_fail`) and respect explicit role override behavior.
- Add tests in the changed package, prefer table-driven cases for branching behavior, and keep tests parallel-safe (`t.Parallel()`).
- Fix root causes; do not suppress errors or bypass failing gates.
- Do NOT run `git commit`. Write only the commit message text to `.ralph/.commit-msg`.
- Keep changes scoped to the selected story and do not start another story in the same turn.

## Quality Gates

Run required quality gates before finishing the turn.
- Run the selected task's DoD command exactly as written in the plan:
- `task-1`: `rg -n "accepted risk|no-action|remediation scope|Findings Baseline" docs/plans/2026-02-25-review-convergence-design.md docs/adr/0004-review-convergence-mode.md docs/plans/2026-02-25-review-feedback-remediation-plan.md`
- `task-2`: `go test ./internal/runner -run 'TestReviewExplicitRoleUsesReviewCompletion|TestReviewJudgeNonZeroDoesNotConverge'`
- `task-3`: `go test ./internal/config -run 'TestLoadBytesAppliesReviewConvergencePartialDefaults|TestLoadBytesReviewConvergenceSingleFieldOverrides'`
- `task-4`: `go test ./internal/runner -run 'TestJudgeContractRejectsUnknownFields|TestJudgeContractRejectsNegativeNewFindings'`
- `task-5`: `go test ./internal/config/... ./internal/config/schema/... && task schema && git diff --exit-code -- schemas/config.schema.json`
- `task-6`: `go test ./internal/runner -run 'TestRunRequiresNonEmptyPromptRun|TestReviewRequiresNonEmptyPromptFiles'`
- `task-7`: `go test ./internal/config ./internal/runner -run 'TestLoadBytesAppliesReviewConvergencePartialDefaults|TestReviewSchedulerRespectsMinReviewsBeforeJudge|TestReviewStableCountResetsOnUnstableJudge'`
- `task-8`: `go test ./internal/config/... ./cmd/ralph/... ./internal/init/... && rg -n "default when omitted|scaffold sample" README.md`
- Run repository-level gates from `AGENTS.md` as applicable: `task fmt`, `task lint`, `task test`, `task build` (and `task schema` whenever config/schema changes).
- Format, lint, and tests must all pass before writing `.ralph/.commit-msg`.
- Verify `git status --short` contains only intended story changes before finishing.
- Fix root causes; do not bypass warnings or relax checks.

## Task Selection Algorithm

1. Read `.ralph/prd.json` to inspect stories and `passes`.
2. Read `.ralph/progress.md` (start with `## Codebase Patterns`) to understand reusable patterns first, then recent progress entries.
3. Select the highest-priority story where:
   - `passes: false`
   - all `deps` stories have `passes: true`
4. Priority rule: choose the lowest numeric story ID among eligible stories.
5. If no story is eligible, report blocked stories and missing dependencies, then end the turn.

## Turn Procedure

1. Pick one eligible story from `.ralph/prd.json`.
2. Read the story details and DoD from the resolved plan file.
3. Implement exactly that one story with TDD.
4. Run quality gates and fix all failures.
5. Write the plan-specified commit message (only the `-m` string) to `.ralph/.commit-msg`.
6. Update `.ralph/prd.json` and set only the completed story's `passes` to `true`.
7. Append progress to `.ralph/progress.md` (append-only; never replace existing content).
8. End your turn.

## Progress Format

Append to `.ralph/progress.md` (never replace existing content):

```md
## [YYYY-MM-DD] - [Story ID]: [Title]
- What was implemented
- DoD verification results
- Learnings:
  - Patterns discovered
  - Gotchas encountered
---
```

## Codebase Patterns

Maintain a `## Codebase Patterns` section near the top of `.ralph/progress.md`.
Add only general, reusable patterns there as they are discovered.
Do not add story-specific details to `## Codebase Patterns`; keep those in dated progress entries.

## Stop Condition

If ALL stories in `.ralph/prd.json` have `passes: true`, reply with EXACTLY:
<promise>COMPLETE</promise>

Otherwise, end normally after writing `.ralph/.commit-msg` and updating tracking files.
