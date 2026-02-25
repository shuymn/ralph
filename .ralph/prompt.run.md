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
The active plan introduces review convergence: command-specific config profiles (`completion.run`/`completion.review`), mode/role-aware runner behavior, and review/judge scheduling and JSON contract checks.
Tech stack and tooling: Go, YAML validation/schema generation, JSON parsing, existing runner/git flows, and Task (`Taskfile.yml`).
<!-- do not edit: plan resolution logic is ralph runtime machinery -->
Resolve the implementation plan file as follows:
1. Read `.ralph/prd.json`.
2. If top-level `plan` is a non-empty string, use that file.
3. Otherwise use the active plan file under `docs/plans/`.

## Rules

- Follow `AGENTS.md` (project policy) and this plan's task-local file boundaries.
- Edit only files listed in the selected task's `Files` section unless a minimal same-scope helper/test adjustment is required for compilation or contract consistency.
- Keep business logic in `internal/*`; CLI surface changes in `main.go` and `cmd/ralph/*`; schema work in `internal/config/schema` and `schemas/config.schema.json`; scaffold/template work in `internal/init/*` and related `README*.md` only when the task requires it.
- Treat design and planning artifacts (`docs/plans/*`, `docs/adr/*`) as reference-only unless the selected story explicitly requires documentation updates.
- Implement exactly ONE story per turn.
- Use TDD (`RED -> GREEN -> REFACTOR`).
- RED must compile and run: a compilation error is not RED. Add minimal scaffolding so the test executes and fails at runtime (for example, a failing assertion or an intentional unimplemented path).
- Write idiomatic Go with small functions and wrap errors with context (`fmt.Errorf("...: %w", err)`).
- Keep external config keys in snake_case YAML tags and preserve required lint annotations.
- For typed runtime/validation errors, implement both `Code() string` and `ExitCode() int`.
- Reuse existing runner/condition semantics (`success()`, `failure()`, `always()`, `changed()`, `on_fail`) rather than inventing new behavior.
- Tests must stay in the changed domain package, prefer table-driven tests for branching behavior, and keep tests parallel-safe (`t.Parallel()`).
- Use the `execute-plan` skill to implement each story. When the Turn Procedure says 'Implement exactly that one story with TDD', invoke `execute-plan` with the corresponding task ID from the plan.
- When tool/sandbox constraints require repository-local output paths via environment variables, keep unintended generated artifacts out of commits.
- If such artifacts are under the repository and not ignored yet, add ignore rules (`.gitignore` for team-shared noise, `.git/info/exclude` for machine-local noise) in the same turn.
- Remove transient artifacts that should not persist after commands finish.
- Do NOT run `git commit`. Write only the commit message text to `.ralph/.commit-msg`.
- Keep changes scoped to the selected story. Do not start another story in the same turn.

## Quality Gates

Run required quality gates before finishing the turn.
- Run the selected task's DoD command exactly as written in the plan:
- `task-1`: `go test ./internal/config/... && task schema && git diff --exit-code -- schemas/config.schema.json`
- `task-2`: `go test ./... -run 'TestRunParsesReviewCommand|TestReviewUsesSharedConfigPath|TestReviewRejectsUnsupportedStrategy'`
- `task-3`: `go test ./internal/runner -run 'TestRunUsesPromptRunPath|TestReviewRequiresPromptFiles|TestRoleCommandFallback|TestRunTailMatchCompatibility'`
- `task-4`: `go test ./internal/runner -run 'TestReviewSchedulerRules|TestReviewSchedulerMaxReviewsBoundary|TestReviewRunIDFormatAndArtifactNaming'`
- `task-5`: `go test ./internal/runner -run 'TestJudgeContractValidation|TestReviewConvergenceStableRounds|TestReviewNonConvergenceReturns23|TestReviewLogsRemainFreeForm'`
- `task-6`: `go test ./internal/runner -run 'TestDryRunRunProfileOutput|TestDryRunReviewProfileOutput'`
- `task-7`: `go test ./internal/init -run 'TestScaffoldCreatesReviewPromptSet|TestScaffoldOmitsLegacyPromptAndReviewsDir|TestScaffoldConfigIncludesRunAndReviewProfiles'`
- Run repository-level quality gates from `AGENTS.md`: `task fmt`, `task lint`, `task test` (and `task schema` whenever config/schema changes).
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
