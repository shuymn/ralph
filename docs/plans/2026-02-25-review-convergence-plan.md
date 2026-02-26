# Review Convergence Implementation Plan

**Source**: `docs/plans/2026-02-25-review-convergence-design.md`
**Trace Pack**: `docs/plans/2026-02-25-review-convergence-plan.trace.md`
**Compose Pack**: `docs/plans/2026-02-25-review-convergence-plan.compose.md`
**Goal**: `ralph run` と `ralph review` を同一 `.ralph/config.yml` で併用し、`review_convergence` 戦略で収斂判定を機械可読に実行できる状態を実現する。
**Architecture**: config は command 別 profile（`completion.run`/`completion.review`）へ移行し、runner は mode/role-aware に拡張する。`review`/`judge` スケジューラと judge JSON 契約を導入しつつ、既存 `tail_match` runtime を維持する。
**Tech Stack**: Go, YAML validation, JSON parsing, existing runner/git flow, Task.

## Task Dependency Graph

`T1 -> T2 -> T3 -> T4 -> T5 -> T6`  
`T1 -> T7`  
`T2 -> T7`

### Task 1: Refactor config model to command profiles

**Satisfied Requirements**: REQ01, REQ02, REQ03, AC08, AC11  
**Design Anchors**: GOAL01, GOAL03, GOAL05, DEC02, DEC06  
**Goal**: `agent.run_command` と command 別 completion profile への移行を実装し、review convergence 設定の妥当性と旧キー拒否を config load 時点で保証する。  
**Dependencies**: none

**Files:**
- Modify: `internal/config/types.go` (新 config 構造と default 定数)
- Modify: `internal/config/validate.go` (strategy/境界値/legacy key 検証)
- Modify: `internal/config/load_test.go` (新キー受理と旧キー拒否のテスト)
- Modify: `internal/config/schema/generator.go` (新 config 形状の schema 反映)
- Modify: `internal/config/schema/generator_test.go` (schema 契約テスト更新)
- Modify: `schemas/config.schema.json` (generated artifact 更新)

**RED**
- `agent.command` や旧 completion 形式を含む設定を受理してしまう場合に失敗するテストを追加する。
- `completion.run.strategy=tail_match` / `completion.review.strategy=review_convergence` 制約や `review_convergence` 境界値検証が欠落すると失敗するテストを追加する。
- Run: `go test ./internal/config ./internal/config/schema -run 'TestLoadBytes|TestSchemaConstraints'`
- Expected: `FAIL with assertion mismatch for legacy acceptance or missing strategy/bounds validation`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- `agent.run_command` 必須化、`agent.review_command` / `agent.judge_command` 任意化、`completion.run` / `completion.review` 追加を実装する。
- review convergence の default と境界値検証、旧キー拒否を実装する。
- schema 生成結果を新 model と整合させる。

**REFACTOR**
- validation エラーメッセージとエラーコードの責務を整理し、config parse/validate の分岐を保守しやすくする。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `agent.run_command` 必須、`completion.run`/`completion.review` の strategy 制約、`review_convergence` 境界値制約が config load で検証される（REQ01, REQ02, REQ03）。
- 旧キー（`agent.command`, 旧 completion 形式）を含む config が exit `22` で失敗する（AC08）。
- `completion.review.strategy=review_convergence` 前提が validation で保証される（AC11）。
- Run: `go test ./internal/config/... && task schema && git diff --exit-code -- schemas/config.schema.json`
- Expected: `PASS`

### Task 2: Add `ralph review` command surface with shared config loading

**Satisfied Requirements**: REQ04, AC11, AC12  
**Design Anchors**: GOAL01, GOAL05, DEC01, DEC02  
**Goal**: CLI に `ralph review` を追加し、`ralph run` と同一 `.ralph/config.yml` を読み込む command surface を確立する。  
**Dependencies**: T1

**Files:**
- Modify: `main.go` (usage と `run`/`review` 分岐)
- Create: `cmd/ralph/review.go` (`RunReview`/`RunReviewDry` entrypoint)
- Modify: `cmd/ralph/run.go` (run/review 共通化しやすい構造へ整理)
- Create: `cmd/ralph/review_test.go` (review command 挙動テスト)
- Create: `main_test.go` (CLI 引数解決テスト)

**RED**
- `ralph review` が未提供、または `.ralph/config.yml` 以外を読む実装になっている場合に失敗するテストを追加する。
- `completion.review.strategy!=review_convergence` の設定でも review を許可してしまう場合に失敗するテストを追加する。
- Run: `go test ./... -run 'TestRunParsesReviewCommand|TestReviewUsesSharedConfigPath|TestReviewRejectsUnsupportedStrategy'`
- Expected: `FAIL with assertion mismatch for missing review subcommand or invalid strategy guard`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- `ralph run` / `ralph review` の双方を同一 config 読み込み経路に統一する。
- review mode 前提 validation（strategy guard）を接続し、違反時に exit `22` を返す。

**REFACTOR**
- command の共通エラーハンドリングと exit code 変換を関数化し、run/review の重複を除去する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `ralph review` は `completion.review.strategy=review_convergence` のときのみ実行可能で、違反時は exit `22` になる（REQ04, AC11）。
- `ralph run` と `ralph review` が同一 `.ralph/config.yml` を参照する（AC12）。
- Run: `go test ./... -run 'TestRunParsesReviewCommand|TestReviewUsesSharedConfigPath|TestReviewRejectsUnsupportedStrategy'`
- Expected: `PASS`

### Task 3: Introduce mode/role command and prompt resolution in runner

**Satisfied Requirements**: REQ05, REQ06, REQ07, AC02, AC06, AC07, AC19  
**Design Anchors**: GOAL01, GOAL02, DEC03, DEC04  
**Goal**: run/review mode で必要な prompt 入力と command 解決を role-aware に実装し、run mode の `tail_match` runtime を回帰させない。  
**Dependencies**: T2

**Files:**
- Modify: `internal/runner/loop.go` (mode-aware 実行分岐)
- Create: `internal/runner/mode_plan.go` (prompt/command 解決モデル)
- Modify: `internal/runner/main_step.go` (role ごとの stdin path 受け渡し)
- Modify: `internal/runner/loop_test.go` (run/review mode 契約テスト)

**RED**
- `run` が `.ralph/prompt.run.md` を使わない、または欠落/空ファイルで失敗しない場合に失敗するテストを追加する。
- `review` が `.ralph/prompt.review.md` / `.ralph/prompt.judge.md` 欠落時に失敗しない場合に失敗するテストを追加する。
- `review_command` / `judge_command` 未指定時に `run_command` fallback しない場合に失敗するテストを追加する。
- Run: `go test ./internal/runner -run 'TestRunUsesPromptRunPath|TestReviewRequiresPromptFiles|TestRoleCommandFallback|TestRunTailMatchCompatibility'`
- Expected: `FAIL with assertion mismatch for prompt path or command resolution rules`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- mode/role ごとに prompt path と command を解決するロジックを実装する。
- prompt 欠落・空ファイルを validation/runtime error（exit `22`）として扱う。
- run mode の completion 挙動（`tail_match`）を維持する。

**REFACTOR**
- path 解決・ファイル存在/空チェック・command fallback を小さなヘルパーに分割し、dry-run と実行系で再利用する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `agent.review_command` / `agent.judge_command` が空のとき `agent.run_command` fallback が使われる（REQ07, AC02）。
- review mode で `.ralph/prompt.review.md` / `.ralph/prompt.judge.md` が欠落している場合は exit `22` で失敗する（REQ06, AC06）。
- run mode で `.ralph/prompt.run.md` が欠落または空の場合は exit `22` で失敗する（AC19）。
- `completion.run.strategy=tail_match` の runtime 停止判定が従来どおり維持される（REQ05, AC07）。
- Run: `go test ./internal/runner -run 'TestRunUsesPromptRunPath|TestReviewRequiresPromptFiles|TestRoleCommandFallback|TestRunTailMatchCompatibility'`
- Expected: `PASS`

### Task 4: Implement review scheduler state and artifact lifecycle

**Satisfied Requirements**: REQ08, REQ09, REQ10, AC01, AC03, AC14  
**Design Anchors**: GOAL02, GOAL03, DEC02, DEC05  
**Goal**: `review`/`judge` スケジューリング規則と runtime state を実装し、`run_id` とレビュー成果物配置を deterministic にする。  
**Dependencies**: T3

**Files:**
- Create: `internal/runner/review_scheduler.go` (role 選択ロジック)
- Create: `internal/runner/review_state.go` (review/judge counters と run_id)
- Create: `internal/runner/review_artifacts.go` (REVIEW/JUDGE path 管理)
- Modify: `internal/runner/loop.go` (review mode ループ統合)
- Create: `internal/runner/review_scheduler_test.go` (スケジュール規則テスト)

**RED**
- `min_reviews` 未満で judge が走る、`judge_every` 間隔が守られない、`max_reviews` 境界で追加 review を実行してしまう場合に失敗するテストを追加する。
- `run_id` が UTC `YYYYMMDDTHHMMSSZ` 形式でない、または `REVIEW_XXXX.md` / `JUDGE_XXXX.json` 命名規約を満たさない場合に失敗するテストを追加する。
- Run: `go test ./internal/runner -run 'TestReviewSchedulerRules|TestReviewSchedulerMaxReviewsBoundary|TestReviewRunIDFormatAndArtifactNaming'`
- Expected: `FAIL with assertion mismatch for scheduling order or artifact naming`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- review/judge role 選択ルール（`min_reviews`, `judge_every`, `max_reviews`）を実装する。
- `review_count`, `reviews_since_judge`, `judge_count`, `run_id` を state として管理し、artifact path に反映する。

**REFACTOR**
- scheduler の状態遷移を pure function 化し、loop 分岐の複雑度を下げる。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `judge_every` ごとに judge role が実行される（REQ08, AC03）。
- `review_count` と `run_id`（UTC `YYYYMMDDTHHMMSSZ`）が正しく管理され、artifact ディレクトリ/ファイル名に反映される（REQ09, REQ10, AC14）。
- `completion.review.strategy=review_convergence` 時に `ralph review` が review/judge role を自動スケジュールする（AC01）。
- Run: `go test ./internal/runner -run 'TestReviewSchedulerRules|TestReviewSchedulerMaxReviewsBoundary|TestReviewRunIDFormatAndArtifactNaming'`
- Expected: `PASS`

### Task 5: Enforce judge JSON contract and convergence completion semantics

**Satisfied Requirements**: REQ11, REQ12, REQ16, AC04, AC05, AC13, AC15  
**Design Anchors**: GOAL02, GOAL04, DEC05, DEC06  
**Goal**: judge JSON を strict parse して安定判定を実装し、収斂成功/非収斂終了の exit code 契約を確定する。  
**Dependencies**: T4

**Files:**
- Create: `internal/runner/judge_contract.go` (JSON decode/validate)
- Create: `internal/runner/review_completion.go` (stable_count 判定)
- Modify: `internal/runner/loop.go` (review mode completion 接続)
- Create: `internal/runner/judge_contract_test.go` (契約テスト)
- Modify: `internal/runner/loop_test.go` (収斂/非収斂の exit code テスト)

**RED**
- `signal` / `new_findings` / `new_finding_keys` 欠落・型不一致を受理してしまう場合に失敗するテストを追加する。
- `stable_rounds` 連続安定 + `min_reviews` 条件を満たしても exit `0` にならない、または `max_reviews` 到達時に未収斂でも exit `23` にならない場合に失敗するテストを追加する。
- review/judge の stderr を固定キーで強制してしまう回帰を検知するテストを追加する。
- Run: `go test ./internal/runner -run 'TestJudgeContractValidation|TestReviewConvergenceStableRounds|TestReviewNonConvergenceReturns23|TestReviewLogsRemainFreeForm'`
- Expected: `FAIL with assertion mismatch for judge contract parsing or exit behavior`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- judge JSON 必須キー/型の strict parse を実装し、失敗時は exit `22` とする。
- `signal == completion.review.signal` かつ `new_findings == 0` で stable 判定、非一致時 reset を実装する。
- `min_reviews` と `stable_rounds` の AND 条件で成功終了、`max_reviews` 未収斂で exit `23` を返す。
- stderr ログ形式の固定契約は導入しない。

**REFACTOR**
- judge parse error と completion decision error の分類を整理し、異常系ログの可読性を上げる。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `stable_rounds` 連続で `signal == completion.review.signal` かつ `new_findings == 0` が成立し、かつ `min_reviews` 以上のとき exit `0` になる（REQ12, AC04）。
- `review_count == max_reviews` でも停止条件未達なら exit `23` を返す（REQ12, AC05）。
- `JUDGE_n.json` は `signal` / `new_findings` / `new_finding_keys` 必須で、欠落/型不一致は exit `22` となる（REQ11, AC13）。
- `review`/`judge` の stderr ログは v1 で固定キーを必須化しない（REQ16, AC15）。
- Run: `go test ./internal/runner -run 'TestJudgeContractValidation|TestReviewConvergenceStableRounds|TestReviewNonConvergenceReturns23|TestReviewLogsRemainFreeForm'`
- Expected: `PASS`

### Task 6: Extend dry-run output for run/review profiles

**Satisfied Requirements**: REQ13, REQ14, AC09, AC10  
**Design Anchors**: GOAL01, GOAL05, DEC02, DEC03, DEC04, DEC05  
**Goal**: `ralph run --dry-run` と `ralph review --dry-run` で mode ごとの execution plan を差分なく可視化する。  
**Dependencies**: T5

**Files:**
- Modify: `internal/runner/dryrun.go` (mode-aware 出力)
- Modify: `internal/runner/dryrun_test.go` (run/review dry-run 出力テスト)
- Modify: `cmd/ralph/run.go` (run dry-run の profile 指定)
- Modify: `cmd/ralph/review.go` (review dry-run の profile 指定)

**RED**
- run dry-run が `completion.run` と `prompt.run.md` を表示しない、または review dry-run が role command 解決/収斂パラメータ/JSON 契約を表示しない場合に失敗するテストを追加する。
- Run: `go test ./internal/runner -run 'TestDryRunRunProfileOutput|TestDryRunReviewProfileOutput'`
- Expected: `FAIL with assertion mismatch for missing dry-run profile fields`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- run dry-run に `completion.run` profile 情報と pre/post 実行計画を表示する。
- review dry-run に command 解決結果、`min_reviews/max_reviews/judge_every/stable_rounds`、prompt path、judge JSON 契約を表示する。

**REFACTOR**
- dry-run rendering の mode 共通部分と mode 固有部分を分離し、将来 profile 拡張時の差分を局所化する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `ralph run --dry-run` が `tail_match` profile（strategy/signal/tail_lines, `agent.run_command`, `prompt.run.md`, pre/post 計画）を表示する（REQ13, AC09）。
- `ralph review --dry-run` が `review_convergence` profile（role command 解決結果、収斂パラメータ、review/judge prompt path、judge JSON 契約）を表示する（REQ14, AC10）。
- Run: `go test ./internal/runner -run 'TestDryRunRunProfileOutput|TestDryRunReviewProfileOutput'`
- Expected: `PASS`

### Task 7: Update `ralph init` scaffolding for review convergence defaults

**Satisfied Requirements**: REQ15, AC16, AC17, AC18  
**Design Anchors**: GOAL01, GOAL05, DEC07  
**Goal**: `ralph init` の生成物を review convergence 前提に更新し、初期生成時点で run/review を同一 config で使える状態にする。  
**Dependencies**: T2

**Files:**
- Modify: `internal/init/scaffold.go` (生成対象ファイルの更新)
- Modify: `internal/init/scaffold_test.go` (生成/非生成契約テスト更新)
- Modify: `internal/init/templates/config.tmpl` (run/review profile を含む最小 config)
- Create: `internal/init/templates/prompt.run.tmpl` (run prompt 雛形)
- Create: `internal/init/templates/prompt.review.tmpl` (review prompt 雛形)
- Create: `internal/init/templates/prompt.judge.tmpl` (judge prompt 雛形)
- Modify: `README.md` (init 出力一覧と config 例更新)
- Modify: `README.ja.md` (同上)

**RED**
- `ralph init` が `prompt.run/review/judge` を生成しない、または `prompt.md` / `.ralph/reviews/` を生成してしまう場合に失敗するテストを追加する。
- `config.yml` が `agent.run_command` と `completion.run` / `completion.review` を含まない、または `review_command` / `judge_command` を初期出力してしまう場合に失敗するテストを追加する。
- Run: `go test ./internal/init -run 'TestScaffoldCreatesReviewPromptSet|TestScaffoldOmitsLegacyPromptAndReviewsDir|TestScaffoldConfigIncludesRunAndReviewProfiles'`
- Expected: `FAIL with assertion mismatch for scaffold file set or config template content`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- init 生成対象を `prompt.run.md` / `prompt.review.md` / `prompt.judge.md` に切り替える。
- `config.tmpl` を run/review profile 前提の最小構成へ更新し、`review_command` / `judge_command` は省略する。
- `prompt.md` と `.ralph/reviews/` は生成しない。

**REFACTOR**
- scaffold template spec をデータ駆動化し、将来ファイル追加/削除時の変更点を 1 箇所に集約する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `ralph init` は `.ralph/config.yml`, `.ralph/prompt.run.md`, `.ralph/prompt.review.md`, `.ralph/prompt.judge.md`, `.ralph/prd.json`, `.ralph/progress.md` を生成する（REQ15, AC16）。
- `ralph init` は `.ralph/prompt.md` と `.ralph/reviews/` を生成しない（AC17）。
- 生成される `config.yml` は `agent.run_command` と `completion.run` / `completion.review` を含み、`review_command` / `judge_command` は省略する（AC18）。
- Run: `go test ./internal/init -run 'TestScaffoldCreatesReviewPromptSet|TestScaffoldOmitsLegacyPromptAndReviewsDir|TestScaffoldConfigIncludesRunAndReviewProfiles'`
- Expected: `PASS`

## Checkpoint Summary

- Alignment Verdict: PASS
- Forward Fidelity: PASS
- Reverse Fidelity: PASS
- Non-Goal Guard: PASS
- Granularity Guard: PASS
- Trace Pack: `docs/plans/2026-02-25-review-convergence-plan.trace.md`
- Compose Pack: `docs/plans/2026-02-25-review-convergence-plan.compose.md`
- Updated At: `2026-02-25`
