# Review Feedback Remediation Plan

**Source**: `In-document Findings Baseline (this file)`
**Trace Pack**: `docs/plans/2026-02-25-review-feedback-remediation-plan.trace.md`
**Compose Pack**: `docs/plans/2026-02-25-review-feedback-remediation-plan.compose.md`
**Goal**: 受理済み除外事項（run_id秒粒度、init最小構成）を明示した上で、有効なレビュー指摘を fail-fast と回帰テストで収束させる。
**Remediation Scope**: findings status は `fix` / `accepted` / `no-action` を正規語彙として扱い、task 単位の判定に再利用する。
**Architecture**: `review` 実行時の完了判定は scheduler 有無ではなく mode 契約で評価する。config/judge 契約は可能な限り load/parse 時点で fail-closed とし、README/設計記録を実装と同期させる。
**Tech Stack**: Go, YAML validation, JSON decode, existing runner/config schema generator, Task.

## Task Dependency Graph

`T1 -> T2 -> T4 -> T8`
`T1 -> T3 -> T5 -> T8`
`T1 -> T6`
`T3 -> T7`

## Findings Baseline (In-Repo Snapshot)

- F01: `ModeReview + explicit role` で review completion 契約を外れる可能性 | status: `fix`
- F02: judge 非0終了でも収束成功にカウントされ得る | status: `fix`
- F03: `review_convergence` 部分指定が未指定フィールド `0` で validation failure になる | status: `fix`
- F04: judge JSON の未知フィールドが受理される | status: `fix`
- F05: `completion.run.tail_lines <= 0` が load 時に reject されない | status: `fix`
- F06: prompt empty-file reject の回帰テスト不足 | status: `fix`
- F07: `review_convergence` 境界ケース（judge発火順序/stable reset）の回帰テスト不足 | status: `fix`
- F08: `agent.run_command` 必須の直接回帰テスト不足 | status: `fix`
- F09: CLI wrapper（run/review）の回帰テスト不足 | status: `fix`
- F10: scaffold 再実行時の既存 prompt/prd 保護テスト不足 | status: `fix`
- F11: README の `git.fallback_no_gpg_sign` 説明が曖昧（default と scaffold sample の区別） | status: `fix`
- F12: run_id 秒粒度リスクは accepted risk（本計画では変更しない） | status: `accepted`
- F13: init quality gate 指摘は no-action（最小 scaffold 方針を維持） | status: `no-action`

### Task 1: Scope Freeze and Design Record Sync

**Satisfied Requirements**: REQ12, REQ13, AC12, AC13  
**Design Anchors**: GOAL04, DEC01, DEC05, DEC06  
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`  
**Goal**: 受理済み/非対応項目を明文化し、以後の実装タスクが不要なスコープ拡張（run_id変更、init quality gate 強制）を起こさないようにする。  
**Dependencies**: none

**Files:**
- Modify: `docs/plans/2026-02-25-review-convergence-design.md` (remediation前提と非対応事項の追記)
- Modify: `docs/adr/0004-review-convergence-mode.md` (運用上の accepted risk と補修方針追記)
- Modify: `docs/plans/2026-02-25-review-feedback-remediation-plan.md` (Findings Baseline の status 明記: fix/accepted/no-action)

**RED**
- remediation 対象外（run_id秒粒度、init quality gate 強制）が未定義のまま次タスクへ進めると失敗するチェックを追加する。
- Run: `rg -n "run_id.*accepted|quality gate.*no-action|Findings Baseline|F12|F13" docs/plans/2026-02-25-review-convergence-design.md docs/adr/0004-review-convergence-mode.md docs/plans/2026-02-25-review-feedback-remediation-plan.md`
- Expected: `FAIL with missing remediation scope markers`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- 受理済み/非対応項目の rationale と境界を設計文書に反映する。
- `Findings Baseline` で finding 単位の対応ステータス（fix/accepted/no-action）を維持管理する。

**REFACTOR**
- 用語（accepted / no-action / fix）を統一し、以降 task が同一語彙を参照できるよう整理する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- run_id秒粒度は「accepted risk」で明示され、実装変更対象外であることが文書化される（REQ12, AC12）。
- init quality gate 未追加は「最小 scaffold 方針により no-action」で明示される（REQ13, AC13）。
- Run: `rg -n "accepted risk|no-action|remediation scope|Findings Baseline" docs/plans/2026-02-25-review-convergence-design.md docs/adr/0004-review-convergence-mode.md docs/plans/2026-02-25-review-feedback-remediation-plan.md`
- Expected: `PASS`

### Task 2: Fix Review Completion Routing for Explicit Roles

**Satisfied Requirements**: REQ01, REQ02, AC01, AC02  
**Design Anchors**: GOAL01, DEC02, DEC03  
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`  
**Goal**: `ModeReview + explicit role` でも review completion 契約を使い、judge 非0終了を成功収束にカウントしないよう修正する。  
**Dependencies**: T1

**Files:**
- Modify: `internal/runner/loop.go` (mode基準の completion 経路と judge failure handling)
- Modify: `internal/runner/loop_test.go` (manual role + judge non-zero 回帰テスト)

**RED**
- `ModeReview + RoleJudge` が run completion (`tail_match`) 経路に落ちる挙動を再現する失敗テストを追加する。
- judge が valid JSON を出しつつ非0終了した際に `exit 0` へ収束してしまう失敗テストを追加する。
- Run: `go test ./internal/runner -run 'TestReviewExplicitRoleUsesReviewCompletion|TestReviewJudgeNonZeroDoesNotConverge'`
- Expected: `FAIL with assertion mismatch for review completion route or convergence exit`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- review completion 分岐を `mode == review` ベースに切り替える。
- judge 実行失敗時は convergence success に進まないようにする（runtime error もしくは non-converged扱いのどちらかを設計記録に一致させる）。

**REFACTOR**
- `completeModeIteration` と `completeReviewIteration` の責務を整理し、mode/role/scheduler の責務境界を明確化する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `ModeReview + RoleJudge` で review completion 契約が必ず適用される（REQ01, AC01）。
- judge 非0終了は `exit 0` 収束に寄与しない（REQ02, AC02）。
- Run: `go test ./internal/runner -run 'TestReviewExplicitRoleUsesReviewCompletion|TestReviewJudgeNonZeroDoesNotConverge'`
- Expected: `PASS`

### Task 3: Fix Partial Defaulting in `review_convergence`

**Satisfied Requirements**: REQ03, AC03  
**Design Anchors**: GOAL02, DEC04  
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`  
**Goal**: `completion.review.review_convergence` の部分指定で未指定フィールドに default を補完し、妥当な部分上書き設定を受理する。  
**Dependencies**: T1

**Files:**
- Modify: `internal/config/validate.go` (field-by-field default fill)
- Modify: `internal/config/load_test.go` (部分指定 default merge テスト)

**RED**
- `min_reviews` のみ指定した config が load 時に失敗することを再現する失敗テストを追加する。
- `max_reviews` のみ指定した config でも同様に失敗することを再現する失敗テストを追加する。
- Run: `go test ./internal/config -run 'TestLoadBytesAppliesReviewConvergencePartialDefaults|TestLoadBytesReviewConvergenceSingleFieldOverrides'`
- Expected: `FAIL with validation error for omitted sibling fields`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- `min_reviews/max_reviews/judge_every/stable_rounds` の各フィールド単位で default 補完する。
- 既存の下限・相互制約（`max >= min` など）を維持する。

**REFACTOR**
- default 補完と validation 境界を明確化し、将来の profile 拡張時に同種バグを防ぎやすい構造へ整理する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `review_convergence` の部分指定が load/validate を通過し、未指定フィールドが default 補完される（REQ03, AC03）。
- Run: `go test ./internal/config -run 'TestLoadBytesAppliesReviewConvergencePartialDefaults|TestLoadBytesReviewConvergenceSingleFieldOverrides'`
- Expected: `PASS`

### Task 4: Enforce Strict Judge JSON Contract

**Satisfied Requirements**: REQ04, AC04  
**Design Anchors**: GOAL02, DEC03  
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`  
**Goal**: judge JSON を strict decode し、未知フィールドを契約違反として reject する。  
**Dependencies**: T2

**Files:**
- Modify: `internal/runner/judge_contract.go` (strict decoder with unknown-field rejection)
- Modify: `internal/runner/judge_contract_test.go` (unknown field / negative new_findings テスト)

**RED**
- judge JSON の未知フィールドを受理してしまう失敗テストを追加する。
- `new_findings: -1` を reject できない場合の失敗テストを追加する。
- Run: `go test ./internal/runner -run 'TestJudgeContractRejectsUnknownFields|TestJudgeContractRejectsNegativeNewFindings'`
- Expected: `FAIL with parser accepting unknown keys or negative count`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- judge artifact parse を strict 化し、未知キーを runtime error にする。
- 負値 `new_findings` の rejection を回帰テストで固定する。

**REFACTOR**
- judge contract エラー文言を統一し、利用者が原因を切り分けやすいログへ整理する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- unknown fields を含む judge JSON は reject される（REQ04, AC04）。
- `new_findings < 0` は確実に reject される（REQ04, AC04）。
- Run: `go test ./internal/runner -run 'TestJudgeContractRejectsUnknownFields|TestJudgeContractRejectsNegativeNewFindings'`
- Expected: `PASS`

### Task 5: Validate `completion.run.tail_lines` at Load Time

**Satisfied Requirements**: REQ05, AC05  
**Design Anchors**: GOAL02, DEC04  
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`  
**Goal**: `tail_lines <= 0` を config load/validation で reject し、runtime 到達前に設定不正を検知する。  
**Dependencies**: T3

**Files:**
- Modify: `internal/config/validate.go` (`tail_lines >= 1` validation)
- Modify: `internal/config/load_test.go` (non-positive tail_lines test)
- Modify: `internal/config/schema/generator.go` (tail_lines minimum patch)
- Modify: `internal/config/schema/generator_test.go` (minimum assertion)
- Modify: `schemas/config.schema.json` (generated artifact)

**RED**
- `tail_lines: -1` を受理してしまう失敗テストを追加する。
- schema に `tail_lines.minimum=1` が無い場合に失敗するテストを追加する。
- Run: `go test ./internal/config ./internal/config/schema -run 'TestLoadBytesRejectsNonPositiveTailLines|TestSchemaConstraints'`
- Expected: `FAIL with missing validation/schema minimum`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- config validation で `completion.run.tail_lines >= 1` を強制する。
- schema generator に同制約を反映し、generated schema を更新する。

**REFACTOR**
- completion profile validation の責務を整理し、run/review で fail-fast ポリシーの一貫性を保つ。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `tail_lines <= 0` が load 時点で exit `22` 相当エラーになる（REQ05, AC05）。
- schema generated artifact が `tail_lines.minimum=1` を含む（REQ05, AC05）。
- Run: `go test ./internal/config/... ./internal/config/schema/... && task schema && git diff --exit-code -- schemas/config.schema.json`
- Expected: `PASS`

### Task 6: Add Prompt Empty-File Regression Coverage

**Satisfied Requirements**: REQ06, AC06  
**Design Anchors**: GOAL03, DEC02  
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`  
**Goal**: `prompt.run/review/judge` の empty-file reject パスをテストで固定し、missing だけでなく blank も回帰検知可能にする。  
**Dependencies**: T1

**Files:**
- Modify: `internal/runner/loop_test.go` (blank prompt table cases)

**RED**
- `prompt.run.md` / `prompt.review.md` / `prompt.judge.md` が空でも通ってしまう失敗テストを追加する。
- Run: `go test ./internal/runner -run 'TestRunRequiresNonEmptyPromptRun|TestReviewRequiresNonEmptyPromptFiles'`
- Expected: `FAIL with missing errPromptFileEmpty assertions`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- 既存 `requirePrompt` 契約に沿って empty-file ケースをテストへ追加する。

**REFACTOR**
- prompt 前提テストの table 共有化で重複を削減し、mode 別差分を明確化する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- run/review/judge の空 prompt がすべて runtime error として検知される（REQ06, AC06）。
- Run: `go test ./internal/runner -run 'TestRunRequiresNonEmptyPromptRun|TestReviewRequiresNonEmptyPromptFiles'`
- Expected: `PASS`

### Task 7: Expand Review-Convergence Boundary Coverage

**Satisfied Requirements**: REQ07, AC07  
**Design Anchors**: GOAL03, DEC04  
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`  
**Goal**: `review_convergence` の境界条件（部分指定成功、`min_reviews > judge_every`、stable reset）の回帰網を補強する。  
**Dependencies**: T3

**Files:**
- Modify: `internal/config/load_test.go` (partial override success cases)
- Modify: `internal/runner/review_scheduler_test.go` (`min_reviews > judge_every` guard case)
- Modify: `internal/runner/loop_test.go` または `internal/runner/review_scheduler_test.go` (stable reset case)

**RED**
- 部分指定補完、judge 発火順序、stable reset のいずれかが崩れても検知できない状態を失敗テストで再現する。
- Run: `go test ./internal/config ./internal/runner -run 'TestLoadBytesAppliesReviewConvergencePartialDefaults|TestReviewSchedulerRespectsMinReviewsBeforeJudge|TestReviewStableCountResetsOnUnstableJudge'`
- Expected: `FAIL with missing boundary assertions`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- 境界条件を明示する table-driven tests を追加する。

**REFACTOR**
- role スケジューリングと convergence 検証のテストデータ構造を統一し、保守性を上げる。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- 部分指定 default 補完・judge 発火順序・stable reset が回帰テストで固定される（REQ07, AC07）。
- Run: `go test ./internal/config ./internal/runner -run 'TestLoadBytesAppliesReviewConvergencePartialDefaults|TestReviewSchedulerRespectsMinReviewsBeforeJudge|TestReviewStableCountResetsOnUnstableJudge'`
- Expected: `PASS`

### Task 8: Close Remaining Coverage and Documentation Gaps

**Satisfied Requirements**: REQ08, REQ09, REQ10, REQ11, AC08, AC09, AC10, AC11  
**Design Anchors**: GOAL03, GOAL04, DEC04, DEC06  
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`  
**Goal**: low-severity の未対応項目（config/CLI/init/doc）を一括で閉じ、レビュー残件をゼロにする。  
**Dependencies**: T4, T5

**Files:**
- Modify: `internal/config/load_test.go` (`agent.run_command` required 直接テスト)
- Modify: `cmd/ralph/review_test.go` / `main_test.go` / `cmd/ralph/*_test.go` (run/review wrapper coverage)
- Modify: `internal/init/scaffold_test.go` (既存 prompt/prd preserve の再実行テスト)
- Modify: `README.md` (fallback_no_gpg_sign の default と scaffold sample の説明明確化)

**RED**
- `agent.run_command` 必須の直接検証、CLI wrapper、scaffold preserve、README 説明整合の欠落を失敗テスト/チェックで再現する。
- Run: `go test ./internal/config ./cmd/ralph ./internal/init -run 'TestLoadBytesRejectsMissingRunCommand|TestRunWrapper|TestScaffoldPreservesExistingPromptAndPRD' && rg -n "default when omitted|scaffold sample" README.md`
- Expected: `FAIL with missing assertions or missing README clarifier`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- 未カバーの低優先指摘をテストとドキュメントで埋める。

**REFACTOR**
- wrapper テストの共通 fixture 化と README の default/explicit サンプル記述の重複整理を行う。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `agent.run_command` 必須、CLI wrapper、scaffold preserve の回帰テストが追加される（REQ08, REQ09, REQ10, AC08, AC09, AC10）。
- `git.fallback_no_gpg_sign` の default と scaffold sample の意味が README で曖昧なく説明される（REQ11, AC11）。
- Run: `go test ./internal/config/... ./cmd/ralph/... ./internal/init/... && rg -n "default when omitted|scaffold sample" README.md`
- Expected: `PASS`

## Checkpoint Summary

- Alignment Verdict: PASS
- Forward Fidelity: PASS
- Reverse Fidelity: PASS
- Non-Goal Guard: PASS
- Granularity Guard: PASS
- Temporal Completeness Guard: PASS
- TEMP Summary: introduced=0, retired=0, open=0, waived=0
- Trace Pack: `docs/plans/2026-02-25-review-feedback-remediation-plan.trace.md`
- Compose Pack: `docs/plans/2026-02-25-review-feedback-remediation-plan.compose.md`
- Updated At: `2026-02-25`
