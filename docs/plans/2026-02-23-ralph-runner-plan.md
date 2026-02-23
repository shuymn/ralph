# 2026-02-23 Ralph Runner Plan

- Source: [2026-02-23-ralph-runner-design.md](./2026-02-23-ralph-runner-design.md)
- Trace Pack: [2026-02-23-ralph-runner-plan.trace.md](./2026-02-23-ralph-runner-plan.trace.md)
- Compose Pack: [2026-02-23-ralph-runner-plan.compose.md](./2026-02-23-ralph-runner-plan.compose.md)
- Goal: Deliver a Go-native `ralph run` and `ralph init` flow driven by `.ralph/config.yml`, with deterministic loop control, git commit automation, and dry-run visibility.
- Architecture: `cmd/ralph` entrypoints + `internal/init`, `internal/config`, `internal/condition`, `internal/runner`, `internal/git`.
- Tech Stack: Go, YAML decoding, JSON decoding, git CLI invocation.

## Task Dependency Graph

`T1 -> T2 -> T3 -> T4 -> T5`  
`T2 -> T6`  
`T3 -> T6`  
`T4 -> T6`

## Task 1: Bootstrap `ralph init` scaffolding

- Satisfied Requirements: REQ02, AC02, AC03
- Design Anchors: GOAL06, GOAL07, NONGOAL07, DEC01
- Goal: Create `.ralph/` scaffolding with template files and skip-safe behavior for existing files.
- Dependencies: none
- Files: Create `cmd/ralph/init.go`; create `internal/init/scaffold.go`; create `internal/init/templates/config.tmpl`; create `internal/init/templates/prompt.tmpl`; create `internal/init/templates/prd.tmpl`; create `internal/init/templates/progress.tmpl`; create `internal/init/scaffold_test.go`.
- RED: Add tests that fail when `init` does not create all four template outputs, does not print skipped paths on stderr, or misses schema comment injection in `config.yml`.
- GREEN: Implement scaffold creation, `YYYY-MM-DD` templating for `progress.md`, skip-if-exists logging, and schema URL header in config template.
- REFACTOR: Centralize template rendering and file creation error handling for consistent diagnostics.
- DoD: Run `task check`; expected: new-project scaffold passes and existing-file skip path assertions pass.

## Task 2: Add config and `prd.json` load-time validation

- Satisfied Requirements: REQ03, REQ07, AC04, AC07
- Design Anchors: GOAL01, GOAL02, GOAL04, NONGOAL03, NONGOAL04, NONGOAL06
- Goal: Validate runner inputs before execution without adding separate profile modes or subcommands.
- Dependencies: T1
- Files: Create `internal/config/types.go`; create `internal/config/load.go`; create `internal/config/validate.go`; create `internal/prd/validate.go`; create `internal/config/load_test.go`; create `internal/prd/validate_test.go`.
- RED: Add failing tests for invalid step shape (`run` + `uses`, missing `name`), unsupported builtin, invalid `on_fail`, invalid commit mode, and malformed/duplicate/empty story IDs in `prd.json`.
- GREEN: Implement strict structural checks, defaults (`if=success()`, `on_fail=stop_loop`), and `prd.json` constraints (`stories` required, unique IDs, boolean `passes`, string-array `deps`).
- REFACTOR: Separate parse errors from semantic validation errors and normalize error codes for caller mapping.
- DoD: Run `task check`; expected: invalid fixtures return exit-code-mappable validation errors.

## Task 3: Implement expression evaluator for step `if`

- Satisfied Requirements: REQ04, AC06
- Design Anchors: GOAL01, GOAL02, GOAL03, NONGOAL06
- Goal: Support expression-gated step execution with deterministic function support and failure semantics.
- Dependencies: T2
- Files: Create `internal/condition/lexer.go`; create `internal/condition/parser.go`; create `internal/condition/evaluator.go`; create `internal/condition/evaluator_test.go`.
- RED: Add failing tests for supported functions (`always`, `success`, `failure`, `changed`), boolean operators, parentheses, unknown symbols, and `changed()` git command failure mapping.
- GREEN: Implement parser/evaluator and runtime callback for `changed()` (`git status --porcelain`) with error propagation for exit 22 paths.
- REFACTOR: Extract reusable evaluation context object so pre/main/post phases share the same semantics.
- DoD: Run `task check`; expected: expression truth tables and error cases all pass.

## Task 4: Build the core run loop and completion checks

- Satisfied Requirements: REQ01, REQ06, REQ10, AC01, AC08, AC10
- Design Anchors: GOAL01, GOAL03, GOAL04, GOAL05, DEC01, NONGOAL01, NONGOAL02, NONGOAL05
- Goal: Execute pre/main/post loop in Go using static prompt/prd/progress files and enforce completion with cleanup.
- Dependencies: T2, T3
- Files: Create `cmd/ralph/run.go`; create `internal/runner/loop.go`; create `internal/runner/main_step.go`; create `internal/runner/completion.go`; create `internal/runner/tmpfile.go`; create `internal/runner/loop_test.go`.
- RED: Add failing integration-style tests for phase ordering, `stop_loop phase=<phase> step=<step> reason=<reason>` logging, exit-code behavior (0/20/21/22/23), completion mismatch handling, and tmpfile deletion on success/failure/signal.
- GREEN: Implement phase execution engine, main agent invocation with tmpfile capture and tail match, completion gating (`passes=true` for all stories), and guaranteed tmpfile cleanup hooks.
- REFACTOR: Isolate exit-code classification and signal-handling cleanup to reduce branching complexity in loop control.
- DoD: Run `task check`; expected: loop semantics, completion semantics, and cleanup guarantees pass.

## Task 5: Implement builtin `uses: auto_commit`

- Satisfied Requirements: REQ05, REQ09, AC05, AC10
- Design Anchors: GOAL01, GOAL02, DEC01, NONGOAL02
- Goal: Provide deterministic split/together git commit behavior with safe no-op handling.
- Dependencies: T4
- Files: Create `internal/git/auto_commit.go`; create `internal/git/task_id.go`; create `internal/git/auto_commit_test.go`.
- RED: Add failing tests for split-mode staging rules (`.ralph/` first), together-mode all-files commit, `${task_id}` extraction constraints (exactly one `false -> true`), empty staged diff no-op, and `.ralph/.commit-msg` fallback behavior.
- GREEN: Implement git orchestration (`git add -A`, `git restore --staged` as needed), commit message resolution, and failure paths for ambiguous task ID extraction.
- REFACTOR: Wrap git command execution behind an interface to simplify deterministic tests and improve log consistency.
- DoD: Run `task check`; expected: split/together strategies and fallback message flow pass.

## Task 6: Add `ralph run --dry-run` plan output

- Satisfied Requirements: REQ08, AC09
- Design Anchors: GOAL01, GOAL02, GOAL04, GOAL05, NONGOAL06
- Goal: Expose the full execution plan without running commands while keeping validation parity with normal run mode.
- Dependencies: T2, T3, T4
- Files: Modify `cmd/ralph/run.go`; create `internal/runner/dryrun.go`; create `internal/runner/dryrun_test.go`.
- RED: Add failing tests that require dry-run output to include agent command, iteration controls, pre/post steps (`name`, `run/uses`, `if`, `on_fail`), git mode/fallback, completion config, and to assert that no subprocess runs.
- GREEN: Implement dry-run renderer wired to existing validated config/prd/evaluator logic and command suppression.
- REFACTOR: Share run-mode metadata assembly between normal run and dry-run output paths to prevent divergence.
- DoD: Run `task check`; expected: output snapshots match spec and command execution count remains zero.

## Checkpoint Summary

- Alignment Verdict: PASS
- Forward Fidelity: PASS (GOAL/DEC/REQ/AC = 28/28 atoms mapped)
- Reverse Fidelity: PASS (T1-T6 all anchored to REQ and AC)
- Non-Goal Guard: PASS (NONGOAL01-NONGOAL07 explicitly guarded)
- Granularity Guard: PASS (all tasks remain single-commit scope)
- Trace Pack: [2026-02-23-ralph-runner-plan.trace.md](./2026-02-23-ralph-runner-plan.trace.md)
- Compose Pack: [2026-02-23-ralph-runner-plan.compose.md](./2026-02-23-ralph-runner-plan.compose.md)
- Updated At: 2026-02-23
