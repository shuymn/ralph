# Review Feedback Remediation Plan Trace Pack

**Input Baseline**: `docs/plans/2026-02-25-review-feedback-remediation-plan.md` の `Findings Baseline (In-Repo Snapshot)`

## Design Atom Index

### Goals
- GOAL01: `ModeReview` では role override 時も review completion 契約を維持する。
- GOAL02: 設定/契約不正は runtime ではなく load/parse で fail-fast に検知する。
- GOAL03: 回帰テスト網を補強し、再発を防止する。
- GOAL04: 受理済みリスクと no-action 項目を文書化し、スコープ逸脱を防ぐ。

### Non-Goals
- NONGOAL01: run_id 形式（秒粒度 timestamp）を変更しない。
- NONGOAL02: init template に quality gate step を必須追加しない。
- NONGOAL03: 新しい CLI フラグ/モードを追加しない。

### Requirements
- REQ01: `ModeReview + explicit role` でも review completion 経路を使う。
- REQ02: judge 非0終了は convergence success にカウントしない。
- REQ03: `review_convergence` の部分指定で未指定フィールドを default 補完する。
- REQ04: judge JSON の未知フィールドを reject する。
- REQ05: `completion.run.tail_lines <= 0` を config load 時に reject する。
- REQ06: `prompt.run/review/judge` の empty-file reject を回帰テストで固定する。
- REQ07: `review_convergence` 境界（部分指定成功/judge発火順序/stable reset）をテストで固定する。
- REQ08: `agent.run_command` 必須の直接回帰テストを追加する。
- REQ09: CLI wrapper（run/review）の回帰テストを追加する。
- REQ10: scaffold 再実行時の既存 prompt/prd 保護テストを追加する。
- REQ11: `git.fallback_no_gpg_sign` の default と scaffold sample の意味を README で明確化する。
- REQ12: run_id秒粒度リスクを accepted として明示する。
- REQ13: init quality gate 指摘は no-action（最小 scaffold 方針）として明示する。

### Decisions
- DEC01: run_id秒粒度は現時点で accepted risk とする。
- DEC02: review completion 分岐は scheduler state ではなく mode 契約で決める。
- DEC03: judge 契約は strict（未知キー reject、非0終了は成功収束に寄与しない）。
- DEC04: `tail_lines` は fail-closed（validation + schema minimum）を採用する。
- DEC05: init は最小 scaffold 方針を維持し、quality gate step を必須化しない。
- DEC06: README では「default値」と「scaffoldの明示設定例」を区別して説明する。

### Acceptance Criteria
- AC01: `ModeReview + RoleJudge` で review completion 経路が使われる。
- AC02: judge 非0終了で `exit 0` 収束しない。
- AC03: `review_convergence` 部分指定が load 成功し、未指定は default 補完される。
- AC04: judge JSON unknown fields が reject される。
- AC05: `tail_lines <= 0` が load 時に失敗する。
- AC06: empty prompt（run/review/judge）が回帰テストで固定される。
- AC07: `review_convergence` 境界ケース回帰テストが追加される。
- AC08: `agent.run_command` 必須の直接テストが追加される。
- AC09: CLI wrapper の回帰テストが追加される。
- AC10: scaffold 再実行で既存 prompt/prd が保護される。
- AC11: README の fallback_no_gpg_sign 説明が曖昧でない。
- AC12: run_id accepted risk が設計/ADRに記録される。
- AC13: init quality gate no-action が設計/ADRに記録される。

## Decision Trace
- DEC01 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)
- DEC02 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)
- DEC03 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)
- DEC04 -> ADR-0002 (`docs/adr/0002-generate-config-schema-from-go-types.md`)
- DEC05 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)
- DEC06 -> ADR-0002 (`docs/adr/0002-generate-config-schema-from-go-types.md`)

## Design -> Task Trace Matrix
- GOAL01 -> T2
- GOAL02 -> T3, T4, T5
- GOAL03 -> T6, T7, T8
- GOAL04 -> T1, T8
- NONGOAL01 -> no task mapping (guarded by T1 scope freeze)
- NONGOAL02 -> no task mapping (guarded by T1 scope freeze)
- NONGOAL03 -> no task mapping (guarded by task scope)
- REQ01 -> T2
- REQ02 -> T2
- REQ03 -> T3
- REQ04 -> T4
- REQ05 -> T5
- REQ06 -> T6
- REQ07 -> T7
- REQ08 -> T8
- REQ09 -> T8
- REQ10 -> T8
- REQ11 -> T8
- REQ12 -> T1
- REQ13 -> T1
- AC01 -> T2
- AC02 -> T2
- AC03 -> T3
- AC04 -> T4
- AC05 -> T5
- AC06 -> T6
- AC07 -> T7
- AC08 -> T8
- AC09 -> T8
- AC10 -> T8
- AC11 -> T8
- AC12 -> T1
- AC13 -> T1
- DEC01 -> T1
- DEC02 -> T2, T6
- DEC03 -> T2, T4
- DEC04 -> T3, T5, T7, T8
- DEC05 -> T1
- DEC06 -> T1, T8

## Task -> Design Compose Matrix
- T1: REQ12, REQ13, AC12, AC13, GOAL04, DEC01, DEC05, DEC06
- T2: REQ01, REQ02, AC01, AC02, GOAL01, DEC02, DEC03
- T3: REQ03, AC03, GOAL02, DEC04
- T4: REQ04, AC04, GOAL02, DEC03
- T5: REQ05, AC05, GOAL02, DEC04
- T6: REQ06, AC06, GOAL03, DEC02
- T7: REQ07, AC07, GOAL03, DEC04
- T8: REQ08, REQ09, REQ10, REQ11, AC08, AC09, AC10, AC11, GOAL03, GOAL04, DEC04, DEC06

## Temporary Mechanism Trace
- none

## Cross Self-Check

### Forward Fidelity (Design -> Tasks)
- Coverage ratio (`REQ+AC covered / total REQ+AC`): `26/26`
- Coverage ratio (`DEC covered / total DEC`): `6/6`
- Invalid DEC-to-ADR mappings: none
- Missing design atoms: none
- Verdict: PASS

### Reverse Fidelity (Tasks -> Design)
- Orphan tasks (no valid anchors): none
- Tasks missing `REQxx/ACxx` in `Satisfied Requirements`: none
- Tasks referencing unknown design atoms: none
- Alignment verdict: PASS
- Gaps and actions: none

### Non-Goal Guard
- Violations against `NONGOALxx`: none
- Check result: no task changes run_id format or init quality gate mandatory behavior.
- Verdict: PASS

### DoD Semantics Guard
- Tasks with OR-like DoD wording: none
- DoD items missing independent verification: none
- Verdict: PASS

### Granularity Guard
- Tasks too broad for a single coherent change unit: none
- Tasks too fragmented (should be merged): none
- Notes: runner behavior fixes, config/schema fixes, and coverage/docs fixes are split by subsystem.
- Verdict: PASS

### Temporal Completeness Guard
- TEMP entries missing introducing tasks: none
- TEMP entries missing retiring tasks: none
- Retire tasks missing negative fallback-removal verification: none
- Open TEMP entries without waiver metadata: none
- Verdict: PASS

### Round-Trip Gate
- Alignment verdict: PASS
- Required fixes: none
- Checked At: 2026-02-25
