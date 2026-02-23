# Ralph Agent Instructions - [PROJECT_NAME]

<!--
This file is a scaffold for human+LLM co-editing.
Step 1: Replace all [placeholders].
Step 2: Keep only project-specific rules.
Step 3: Remove this comment block when finalized.
-->

## Context

You are running in the root of `[PROJECT_NAME]`.
Loop files are in `.ralph/`.
Primary plan is `[PLAN_PATH]`.

## Non-Negotiable Rules

- Follow `AGENTS.md` and project conventions (`[STYLE_DOCS]` if present).
- Implement exactly ONE story per turn.
- Do NOT run `git commit`; write only the commit message to `.ralph/.commit-msg`.
- Do not edit protected paths: `[PROTECTED_PATHS]`.
- Keep changes minimal and directly scoped to the selected story.

## Quality Gates (must pass before ending the turn)

1. `[FMT_COMMAND]`
2. `[LINT_COMMAND]`
3. `[TEST_COMMAND]`

These gates are agent-facing expectations; runner `post` steps may execute the same commands again for enforcement.

If any gate fails, fix root causes before updating tracking files.

## Task Selection Algorithm

1. Read `.ralph/prd.json`.
2. Read `.ralph/progress.md`.
3. Select the highest-priority story where:
- `passes == false`
- all `deps` stories have `passes == true`
4. Tie-breaker: lowest numeric task ID first.
5. If no story is eligible, report blocked stories with reasons and stop.

## Turn Procedure

1. Pick one eligible story from `.ralph/prd.json`.
2. Read the matching details in `[PLAN_PATH]`.
3. Implement with TDD (`RED -> GREEN -> REFACTOR`).
4. Run quality gates and fix failures.
5. Write commit message text to `.ralph/.commit-msg`.
6. Update `.ralph/prd.json` (`passes: true` for the completed story).
7. Append progress to `.ralph/progress.md`.
8. End the turn. Do not start another story.

## Progress Format

Append to `.ralph/progress.md`:

```md
## [YYYY-MM-DD] - [story-id]: [title]
- What was implemented
- Files changed
- Test / lint / format results
- Learnings:
  - Patterns discovered
  - Gotchas encountered
---
```

## Stop Condition

If ALL stories in `.ralph/prd.json` have `passes: true`, reply with EXACTLY:
`<promise>COMPLETE</promise>`
Note: If you customize `completion.signal` in `.ralph/config.yml`, update the literal above to the exact same value.

Otherwise end normally after writing `.ralph/.commit-msg` and updating tracking files.

## Co-Editing Checklist (for user + LLM)

- [ ] Replace all placeholders (`[PROJECT_NAME]`, `[PLAN_PATH]`, commands, protected paths).
- [ ] Remove rules that do not apply to this repository.
- [ ] Add repository-specific Do/Don't items discovered during the first 2-3 turns.
- [ ] Confirm `.ralph/prd.json` task IDs and dependency graph are valid.
- [ ] Keep this file concise: only persistent rules, no temporary task notes.

## Prompt Hygiene (keep this section)

- This prompt is a living contract. Update when behavior changes.
- Prefer explicit constraints over long prose.
- If a rule is repeatedly ignored, rewrite it with concrete examples.
- Move one-off guidance to `.ralph/progress.md` instead of bloating this file.
