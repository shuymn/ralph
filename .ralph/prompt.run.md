# Ralph Agent Instructions

<!--
Editing guide:
- Keep this prompt concise and persistent; move one-off notes to `.ralph/progress.md`.
- Prefer concrete, testable rules over abstract wording.
- Update Context/Rules/Quality Gates to match the repository, but keep the one-story-per-turn workflow.
- Do not modify Task Selection, Turn Procedure, Progress Format, Codebase Patterns, or Stop Condition — these are ralph runtime machinery.
-->

## Context

You are running in the repository root.
Loop files are in `.ralph/`.
This repository is `ralph`: a Go CLI control plane with a bundled TypeScript gateway execution plane.
- CLI entrypoints: `main.go`, `cmd/ralph/*`.
- Core domains: `internal/config`, `internal/runner`, `internal/gateway`, plus `git`, `condition`, `prd`, `init`.
- Gateway side lives in `gateway/*` (bun/TypeScript) and communicates with Go via stdio JSON-RPC.
- Design records live in `docs/plans/*`; ADRs live in `docs/adr/*`.
<!-- do not edit: plan resolution logic is ralph runtime machinery -->
Resolve the implementation plan file as follows:
1. Read `.ralph/prd.json`.
2. If top-level `plan` is a non-empty string, use that file.
3. Otherwise use the active plan file under `docs/plans/`.

## Rules

- Follow `AGENTS.md` (project policy), and follow repository coding/testing guidance documents when present.
- Use the `execute-plan` skill when implementing each selected story during the Turn Procedure.
- Respect protected or reference-only paths declared in `AGENTS.md` or the selected plan.
- Place new code in source directories defined by the selected plan.
- For the selected story, modify only paths listed under that story's `Files` section unless the plan explicitly requires additional files.
- Treat `docs/plans/*` and `docs/adr/*` as reference-only unless the selected story explicitly includes them.
- Keep business logic inside `internal/*` and preserve domain-driven package boundaries.
- In Go code, prefer small functions and wrap errors with context (`fmt.Errorf(\"...: %w\", err)`).
- For typed runtime/validation errors, implement both `Code() string` and `ExitCode() int`.
- For config/schema changes, keep external YAML keys snake_case and preserve strict unknown-key validation.
- Implement exactly ONE story per turn.
- Use TDD (`RED -> GREEN -> REFACTOR`).
- RED must compile and run: a compilation error is not RED. Add minimal scaffolding so the test executes and fails at runtime (for example, a failing assertion or an intentional unimplemented path).
- Add tests in the same domain package, use table-driven tests for non-trivial branches, and keep tests parallel-safe with `t.Parallel()`.
- When tool/sandbox constraints require repository-local output paths via environment variables, keep unintended generated artifacts out of commits.
- If such artifacts are under the repository and not ignored yet, add ignore rules (`.gitignore` for team-shared noise, `.git/info/exclude` for machine-local noise) in the same turn.
- Remove transient artifacts that should not persist after commands finish.
- Do NOT run `git commit`. Write only the commit message text to `.ralph/.commit-msg`.
- Keep changes scoped to the selected story. Do not start another story in the same turn.

## Quality Gates

Run required quality gates before finishing the turn.
- Use `task` commands as the default interface.
- Run the selected story's DoD command(s) exactly as written in the resolved plan.
- Repository baseline commands: `task fmt`, `task lint`, `task test`, and `task check` when the story affects shared CI/build flow.
- Run `task schema` and verify `git diff --exit-code -- schemas/config.schema.json` for config/schema stories.
- Plan-specific command families include:
  - `go test ./internal/config/... ./internal/config/schema/...`
  - `go test ./internal/gateway -run '<task-specific regex>'`
  - `go test ./internal/runner -run '<task-specific regex>'`
  - `bun test gateway/test/jsonrpc_server.test.ts`
  - `bun test gateway/test/stream_normalizer.test.ts && ! rg -n 'tail_match|review_convergence|completeReview|exit code' gateway/src`
  - `task test:protocol`
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
