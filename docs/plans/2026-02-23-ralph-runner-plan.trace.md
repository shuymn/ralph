# 2026-02-23 Ralph Runner Plan Trace Pack

## Design Atom Index

### Goals

- GOAL01: `ralph run` reads `.ralph/config.yml` and executes pre/main/post directly in Go.
- GOAL02: Project-specific behavior is captured in YAML configuration.
- GOAL03: Keep one loop iteration to one agent invocation.
- GOAL04: Keep `prd.json` minimal (`deps`, `passes`, and related metadata).
- GOAL05: Use a static `.ralph/prompt.md` file.
- GOAL06: `ralph init` scaffolds `.ralph/config.yml`, `.ralph/prompt.md`, `.ralph/prd.json`, `.ralph/progress.md`.
- GOAL07: Config uses remote JSON schema reference; no local schema export.

### Non-Goals

- NONGOAL01: No plan extraction from `plan.md` for per-task prompt generation.
- NONGOAL02: No built-in distillation/knowledge-merge workflow in runner.
- NONGOAL03: No strict DSL expansion for `prd.json`.
- NONGOAL04: No profile switching mode (minimal/standard) in v1.
- NONGOAL05: No shell script generation path (`ralph generate`) in v1.
- NONGOAL06: No extra `validate` subcommand in v1.
- NONGOAL07: No schema artifact expansion into `.ralph/`.

### Decisions

- DEC01: Choose Go runner execution over shell generation (`docs/adr/0001-go-runner-over-shell-generation.md`).

### Requirements

- REQ01: Run loop executes pre/main/post by reading `.ralph/config.yml`.
- REQ02: `init` creates templates with date rendering and skip-safe behavior.
- REQ03: Step shape validation (`name`, exactly one of `run`/`uses`, defaults, builtin restriction).
- REQ04: `if` evaluator supports required functions/operators and `changed()` via git status.
- REQ05: `auto_commit` supports split/together and task-id aware messaging.
- REQ06: Completion uses tail-match signal and all-stories-passing gate.
- REQ07: `prd.json` validation enforces required story schema invariants.
- REQ08: `--dry-run` prints execution plan and performs same validations without execution.
- REQ09: Runtime git behavior and stop-loop logging are deterministic.
- REQ10: Tmpfile lifecycle is cleaned on normal, error, and signal exits.

### Acceptance Criteria

- AC01: Go-native pre/main/post loop from `.ralph/config.yml`.
- AC02: `init` scaffolds files, skips existing files, and logs skip paths.
- AC03: Initial config stays minimal and includes schema directive.
- AC04: Step validation/default semantics are enforced.
- AC05: `auto_commit` split/together behavior is correct.
- AC06: `if` evaluator behavior and rejection paths are correct.
- AC07: `prd.json` validation blocks invalid story definitions before loop start.
- AC08: Tail-match completion and all-story pass checks gate success.
- AC09: Dry-run output contains full plan and runs no commands.
- AC10: Exit codes and cleanup behavior are consistent.

## Decision Trace

- DEC01 -> ADR-0001 (`docs/adr/0001-go-runner-over-shell-generation.md`)

## Design -> Task Trace Matrix

- GOAL01 -> T2, T3, T4, T6
- GOAL02 -> T2, T3, T5, T6
- GOAL03 -> T3, T4
- GOAL04 -> T2, T4, T6
- GOAL05 -> T1, T4, T6
- GOAL06 -> T1
- GOAL07 -> T1, T2
- DEC01 -> T1, T4, T5
- REQ01 -> T4
- REQ02 -> T1
- REQ03 -> T2
- REQ04 -> T3
- REQ05 -> T5
- REQ06 -> T4
- REQ07 -> T2
- REQ08 -> T6
- REQ09 -> T5
- REQ10 -> T4
- AC01 -> T4
- AC02 -> T1
- AC03 -> T1
- AC04 -> T2
- AC05 -> T5
- AC06 -> T3
- AC07 -> T2
- AC08 -> T4
- AC09 -> T6
- AC10 -> T4, T5
- NONGOAL01 -> T4 guard
- NONGOAL02 -> T4, T5 guard
- NONGOAL03 -> T2 guard
- NONGOAL04 -> T2 guard
- NONGOAL05 -> T4 guard
- NONGOAL06 -> T2, T6 guard
- NONGOAL07 -> T1 guard

## Task -> Design Compose Matrix

- T1: GOAL05, GOAL06, GOAL07, DEC01, REQ02, AC02, AC03, NONGOAL07
- T2: GOAL01, GOAL02, GOAL04, GOAL07, REQ03, REQ07, AC04, AC07, NONGOAL03, NONGOAL04, NONGOAL06
- T3: GOAL01, GOAL02, GOAL03, REQ04, AC06, NONGOAL06
- T4: GOAL01, GOAL03, GOAL04, GOAL05, DEC01, REQ01, REQ06, REQ10, AC01, AC08, AC10, NONGOAL01, NONGOAL02, NONGOAL05
- T5: GOAL01, GOAL02, DEC01, REQ05, REQ09, AC05, AC10, NONGOAL02
- T6: GOAL01, GOAL02, GOAL04, GOAL05, REQ08, AC09, NONGOAL06

## Cross Self-Check

### Forward Fidelity (Design -> Tasks)

- Coverage ratio (GOAL + DEC + REQ + AC): 28/28 (100%)
- Coverage ratio (NONGOAL guard): 7/7 (100%)
- Invalid mappings: none
- Missing mappings: none
- Verdict: PASS

### Reverse Fidelity (Tasks -> Design)

- Orphan tasks: none
- Tasks missing REQ anchors: none
- Tasks missing AC anchors: none
- Tasks referencing unknown atoms: none
- Verdict: PASS

### Non-Goal Guard

- Violations: none
- Notes: all non-goals are mapped as explicit scope guards in at least one task.
- Verdict: PASS

### Granularity Guard

- Overly broad tasks: none
- Overly fragmented tasks: none
- Notes: each task is scoped to one subsystem and sized for a single commit.
- Verdict: PASS

### Alignment Verdict

- Verdict: PASS
- Gaps/Actions: none
- Checked At: 2026-02-23
