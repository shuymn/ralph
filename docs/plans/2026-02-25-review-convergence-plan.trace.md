# Review Convergence Plan Trace Pack

## Design Atom Index

### Goals

- GOAL01: command 別 completion profile（`completion.run` / `completion.review`）を同一 config で共存させる。
- GOAL02: runner に `review` / `judge` の 2 role 実行モデルを導入する。
- GOAL03: 収斂制御パラメータ（`min_reviews`, `max_reviews`, `judge_every`, `stable_rounds`）を設定可能にする。
- GOAL04: 既存 exit code 体系を維持し、非収斂上限到達は `23` を再利用する。
- GOAL05: `ralph run` と `ralph review` を同一 `.ralph/config.yml` で切り替えて実行可能にする。

### Non-Goals

- NONGOAL01: 1 回の `ralph run` で `tail_match` と `review_convergence` を同時評価しない。
- NONGOAL02: 既存 pre/main/post モデルを廃止しない。
- NONGOAL03: runner にレビュー内容の高度自然言語解析を強制しない。
- NONGOAL04: v1 で reviewer 並列実行を導入しない。

### Decisions

- DEC01: 実行基盤は Go runner（config 直接実行）を継続する。
- DEC02: command 別 completion profile と `ralph run` / `ralph review` の分離 command surface を採用する。
- DEC03: prompt は command/role 別固定 path（`prompt.run.md`, `prompt.review.md`, `prompt.judge.md`）に分離する。
- DEC04: role command 解決は `review_command`/`judge_command` を優先し、未指定時は `run_command` fallback とする。
- DEC05: judge 判定は `JUDGE_n.json` を strict parse して `signal` + `new_findings` で収斂判定する。
- DEC06: 非収斂上限到達時は既存 `ExitCodeMaxIterations (23)` を再利用する。
- DEC07: `ralph init` は prompt 3 分割を生成し、`.ralph/prompt.md` と `.ralph/reviews/` は生成しない。

### Requirements

- REQ01: config は `agent.run_command` 必須、`agent.review_command` / `agent.judge_command` 任意をサポートする。
- REQ02: completion は `completion.run` / `completion.review` の command 別 profile 構造を持つ。
- REQ03: validation は strategy 制約と review convergence 境界値制約、および旧キー（`agent.command`・旧 completion 形式）拒否を行う。
- REQ04: CLI は `ralph run` と `ralph review` を提供し、両者とも同一 `.ralph/config.yml` を読む。
- REQ05: run mode は既存 `tail_match` 停止判定と pre/main/post 実行モデルを維持する。
- REQ06: prompt path は run/review/judge で固定し、必要ファイル欠落または空ファイル時は exit `22` とする。
- REQ07: role command は `review_command` / `judge_command` 優先、未指定時 `run_command` fallback で解決する。
- REQ08: review scheduler は `min_reviews`, `judge_every`, `max_reviews` を満たす role 選択規則を実装する。
- REQ09: review runtime state として `review_count`, `reviews_since_judge`, `stable_count`, `judge_count`, `run_id` を保持する。
- REQ10: review/judge artifacts は `.ralph/reviews/<run_id>/REVIEW_XXXX.md` と `JUDGE_XXXX.json` 命名規約で管理する。
- REQ11: `JUDGE_n.json` は `signal` / `new_findings` / `new_finding_keys` を必須キーとして strict parse する。
- REQ12: 収斂完了は `signal` 一致 + `new_findings==0` の連続安定（`stable_rounds`）と `min_reviews` を AND 条件にする。未収斂で `max_reviews` 到達時は exit `23`。
- REQ13: `ralph run --dry-run` は run profile（`tail_match`）と run command/prompt/pre-post 計画を表示する。
- REQ14: `ralph review --dry-run` は review profile（role command 解決、収斂パラメータ、prompt path、judge JSON 契約、judge stdin machine context 契約）を表示する。
- REQ15: `ralph init` は run/review profile を含む config と 3 種 prompt を生成し、`review_command` / `judge_command` は初期出力しない。
- REQ16: review/judge の stderr ログ形式は v1 で固定契約を持たない（JSON 契約対象は judge artifact のみ）。

### Acceptance Criteria

- AC01: `completion.review.strategy=review_convergence` 設定時に `ralph review` が review/judge role を自動スケジュールする。
- AC02: `agent.review_command` / `agent.judge_command` が空の場合に `agent.run_command` が fallback で使われる。
- AC03: `judge_every`（default 2）ごとに judge role が実行される。
- AC04: `stable_rounds`（default 2）連続安定 + `min_reviews`（default 3）以上で exit `0` になる。
- AC05: `review_count == max_reviews`（default 10）で未収斂なら exit `23`。
- AC06: review strategy 有効時に `.ralph/prompt.review.md` / `.ralph/prompt.judge.md` 欠落は exit `22`。
- AC07: `completion.run.strategy=tail_match` の runtime 停止判定は従来どおり。
- AC08: 旧キー（`agent.command`, 旧 completion 形式）を含む config は exit `22`。
- AC09: `ralph run --dry-run` は run profile（strategy/signal/tail_lines, run command, prompt.run path, pre/post 計画）を表示する。
- AC10: `ralph review --dry-run` は review profile（収斂設定、role command 解決、review/judge prompt path、judge JSON 契約、judge stdin machine context 契約）を表示する。
- AC11: `ralph review` は `completion.review.strategy=review_convergence` のときのみ実行可能で、それ以外は exit `22`。
- AC12: `ralph run` と `ralph review` は同一 `.ralph/config.yml` を参照する。
- AC13: `JUDGE_n.json` は `signal` / `new_findings` / `new_finding_keys` 必須で、欠落/型不一致は exit `22`。
- AC14: `run_id` は UTC timestamp 形式 `YYYYMMDDTHHMMSSZ` を使う。
- AC15: `review`/`judge` の stderr ログは v1 で固定フォーマットを強制しない。
- AC16: `ralph init` は `.ralph/config.yml`, `.ralph/prompt.run.md`, `.ralph/prompt.review.md`, `.ralph/prompt.judge.md`, `.ralph/prd.json`, `.ralph/progress.md` を生成する。
- AC17: `ralph init` は `.ralph/prompt.md` と `.ralph/reviews/` を生成しない。
- AC18: `ralph init` の `config.yml` は `agent.run_command` と `completion.run` / `completion.review` を含み、`review_command` / `judge_command` は省略する。
- AC19: `ralph run` 実行時に `.ralph/prompt.run.md` 欠落または空ファイルなら exit `22`。

### Verification Baseline

- Test framework: Go `testing` package（table-driven + `t.Parallel()` 方針）。
- Canonical verification commands:
  - `go test ./internal/config/...`
  - `go test ./internal/runner/...`
  - `go test ./internal/init/...`
  - `task schema && git diff --exit-code -- schemas/config.schema.json`
  - `task check`

## Decision Trace

- DEC01 -> ADR-0001 (`docs/adr/0001-go-runner-over-shell-generation.md`)
- DEC02 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)
- DEC03 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)
- DEC04 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)
- DEC05 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)
- DEC06 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)
- DEC07 -> ADR-0004 (`docs/adr/0004-review-convergence-mode.md`)

## Design -> Task Trace Matrix

- GOAL01 -> T1, T2, T3, T6, T7
- GOAL02 -> T3, T4, T5
- GOAL03 -> T1, T4, T6
- GOAL04 -> T1, T5
- GOAL05 -> T1, T2, T6, T7
- NONGOAL01 -> no task mapping (guarded by mode separation and cross-check)
- NONGOAL02 -> no task mapping (guarded by run-mode compatibility checks)
- NONGOAL03 -> no task mapping (guarded by artifact-contract-only design)
- NONGOAL04 -> no task mapping (guarded by single-process scheduler scope)
- DEC01 -> T2
- DEC02 -> T1, T2, T4, T6
- DEC03 -> T3, T6, T7
- DEC04 -> T3, T6
- DEC05 -> T4, T5, T6
- DEC06 -> T1, T5
- DEC07 -> T7
- REQ01 -> T1
- REQ02 -> T1
- REQ03 -> T1
- REQ04 -> T2
- REQ05 -> T3
- REQ06 -> T3
- REQ07 -> T3
- REQ08 -> T4
- REQ09 -> T4
- REQ10 -> T4
- REQ11 -> T5
- REQ12 -> T5
- REQ13 -> T6
- REQ14 -> T6
- REQ15 -> T7
- REQ16 -> T5
- AC01 -> T4
- AC02 -> T3
- AC03 -> T4
- AC04 -> T5
- AC05 -> T5
- AC06 -> T3
- AC07 -> T3
- AC08 -> T1
- AC09 -> T6
- AC10 -> T6
- AC11 -> T1, T2
- AC12 -> T2
- AC13 -> T5
- AC14 -> T4
- AC15 -> T5
- AC16 -> T7
- AC17 -> T7
- AC18 -> T7
- AC19 -> T3

## Task -> Design Compose Matrix

- T1: GOAL01, GOAL03, GOAL05, DEC02, DEC06, REQ01, REQ02, REQ03, AC08, AC11
- T2: GOAL01, GOAL05, DEC01, DEC02, REQ04, AC11, AC12
- T3: GOAL01, GOAL02, DEC03, DEC04, REQ05, REQ06, REQ07, AC02, AC06, AC07, AC19
- T4: GOAL02, GOAL03, DEC02, DEC05, REQ08, REQ09, REQ10, AC01, AC03, AC14
- T5: GOAL02, GOAL04, DEC05, DEC06, REQ11, REQ12, REQ16, AC04, AC05, AC13, AC15
- T6: GOAL01, GOAL03, GOAL05, DEC02, DEC03, DEC04, DEC05, REQ13, REQ14, AC09, AC10
- T7: GOAL01, GOAL05, DEC03, DEC07, REQ15, AC16, AC17, AC18

## Full Cross Self-Check Evidence

### Forward Fidelity (Design -> Tasks)

- Coverage ratio (`REQ+AC covered / total REQ+AC`): `35/35`
- Coverage ratio (`DEC covered / total DEC`): `7/7`
- `GOALxx` coverage (`covered / total`): `5/5`
- Invalid DEC-to-ADR mappings: none
- Missing design atoms: none
- Evidence notes:
  - 全 `REQ01-REQ16` が `Satisfied Requirements` に出現することを確認。
  - 全 `AC01-AC19` が `Satisfied Requirements` に出現することを確認。
  - 全 `REQ/AC` が少なくとも 1 つの task DoD に明示されることを確認。
- Verdict: PASS

### Reverse Fidelity (Tasks -> Design)

- Orphan tasks (no valid anchors): none
- Tasks missing `REQxx/ACxx` in `Satisfied Requirements`: none
- Tasks referencing unknown design atoms: none
- Reconstructed scope preserves source design goals/decisions/acceptance intent: yes
- Alignment verdict: PASS
- Gaps and actions: none

### Non-Goal Guard

- Violations against `NONGOALxx`: none
- Check result:
  - いずれの task も `NONGOALxx` を design anchor に含まない。
  - command 別 strategy 分離、run-mode 回帰、単一 scheduler 前提により NONGOAL 逸脱を防止。
- Verdict: PASS

### DoD Semantics Guard

- Tasks with OR-like DoD wording: none
- DoD items missing independent verification: none
- Notes:
  - T1-T7 すべてで "All DoD items are mandatory AND conditions" を明示。
  - 各 task の DoD は実行可能コマンドと期待結果（PASS）を持つ。
- Verdict: PASS

### Granularity Guard

- Tasks too broad for a single coherent change unit: none
- Tasks too fragmented (should be merged): none
- Notes:
  - config migration、CLI surface、runner resolution、scheduler、completion contract、dry-run、init scaffolding の 7 単位に分割。
  - 各 task は 1 つの主要サブシステムに閉じた commit-sized 変更単位。
- Verdict: PASS

### Round-Trip Gate

- Alignment verdict: PASS
- Required fixes: none
- Checked At: 2026-02-25
