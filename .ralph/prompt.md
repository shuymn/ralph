# Ralph Agent Instructions

<!--
Editing guide:
- Keep this prompt concise and persistent; move one-off notes to `.ralph/progress.md`.
- Prefer concrete, testable rules over abstract wording.
- Update Context/Rules/Quality Gates to match the repository, but keep the one-story-per-turn workflow.
- Do not modify Task Selection, Turn Procedure, Progress Format, Codebase Patterns, or Stop Condition — these are ralph runtime machinery.
-->

## Context

You are running in the `ralph` repository root.
Loop files are in `.ralph/`.
Primary implementation areas for the selected plan:
- `cmd/ralph`: CLI entrypoints (`run`, `init`).
- `internal/init`: `.ralph/` scaffold and templates.
- `internal/config`: `.ralph/config.yml` loading and validation.
- `internal/prd`: `.ralph/prd.json` validation.
- `internal/condition`: step `if` lexer/parser/evaluator.
- `internal/runner`: loop orchestration, completion checks, dry-run output, tmpfile cleanup.
- `internal/git`: builtin `uses: auto_commit`.
Plan architecture: `cmd/ralph` entrypoints + `internal/init`, `internal/config`, `internal/condition`, `internal/runner`, `internal/git`.
Tech stack: Go with YAML decoding, JSON decoding, and git CLI invocation.
<!-- do not edit: plan resolution logic is ralph runtime machinery -->
Resolve the implementation plan file as follows:
1. Read `.ralph/prd.json`.
2. If top-level `plan` is a non-empty string, use that file.
3. Otherwise use the active plan file under `docs/plans/`.

## Rules

- Follow `AGENTS.md` (project policy), and follow repository coding/testing guidance documents when present.
- Respect protected or reference-only paths declared in `AGENTS.md` or the selected plan.
- Keep changes inside the selected story `Files` scope; avoid unrelated paths.
- Place new code in source directories defined by the selected plan (`cmd/ralph`, `internal/init`, `internal/config`, `internal/prd`, `internal/condition`, `internal/runner`, `internal/git`).
- Add code and tests together (`<name>.go` and `<name>_test.go`).
- Use constants or configuration; never hardcode environment-specific values.
- Fix root causes; do not silence lint or test failures.
- Treat `.ralph/` as runtime state and edit it only when workflow behavior requires it.
- Implement exactly ONE story per turn.
- Use TDD (`RED -> GREEN -> REFACTOR`).
- RED must compile and run: a compilation error is not RED. Add minimal scaffolding so the test executes and fails at runtime (for example, a failing assertion or an intentional unimplemented path).
- Do NOT run `git commit`. Write only the commit message text to `.ralph/.commit-msg`.
- Keep changes scoped to the selected story. Do not start another story in the same turn.
- Use the `execute-plan` skill to implement each story. When the Turn Procedure says 'Implement exactly that one story with TDD', invoke `execute-plan` with the corresponding task ID from the plan.

## Quality Gates

Run required quality gates before finishing the turn.
- Mandatory per selected plan task DoD: run `task check`.
- Also run the repository standard commands as needed: `task fmt`, `task lint`, and `task test`.
- Run `task test:coverage` when coverage evidence is required.
- Run `task deps-verify` when `go.mod` or tool versions change.
- If both `AGENTS.md` and the selected plan task specify commands, run all required commands from both.
- Format, lint, and tests must all pass before writing `.ralph/.commit-msg`.
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
- Files changed
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
