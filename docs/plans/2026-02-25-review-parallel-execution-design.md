# Review Parallel Execution - Design

## Overview

本ドキュメントは `ralph review` の実行モデルを拡張し、`review` を最大限並列化しつつ `judge` を直列で実行する設計を定義する。
同時に、`run` と `review` の責務境界を明確化し、`agent.max_iterations` / `agent.sleep_seconds` を `run` 専用に分離する。

本変更は PR #1 専用スコープ（review 並列化 + runner 整理）であり、CLI ライブラリ導入は含めない。

## Goals

- `completion.review.review_convergence.execution_mode`（`parallel|serial`）を追加し、default を `parallel` にする。
- `parallel` で「初回 `min_reviews` 並列、以降 `judge_every` 並列、各バッチ後に `judge` 直列」を実現する。
- `serial` と `parallel` で失敗ポリシーを統一し、`review`/`judge` 非0終了は fail-fast（exit `22`）にする。
- `review` モードから `agent.max_iterations` / `agent.sleep_seconds` 依存を除去し、`run` 専用責務に戻す。
- artifact とカウントの整合（成功時のみ保存/加算、連番欠番なし）を保証する。

## Non-Goals

- CLI フラグによる mode 切替の追加。
- CLI ライブラリ導入（これは Design Doc #2 / PR #2 で扱う）。
- review/judge のプロンプト仕様や judge JSON 契約キーの拡張。

## Background

現行 `review` は 1 iteration = 1 role の直列実行であり、review 回数が増えるほど所要時間が長くなる。
また現在は `run` と `review` が同一ループ制御を共有しており、`review` でも `agent.max_iterations` / `agent.sleep_seconds` の影響を受ける。

今回の目的は、`review` のスループット改善を主眼にしつつ、runner の責務分離を同時に進めることにある。

## Clarifications

| Question | Answer / Assumption | Impact | Status |
|----------|----------------------|--------|--------|
| 並列化の対象はどこか | `review` のみ並列、`judge` は直列 | スケジューラを「review batch + judge」に再設計する | resolved |
| 並列と直列の切替はどうするか | config のみで切替、CLI フラグは追加しない | `execution_mode` を config に追加する | resolved |
| default はどちらか | `parallel` を default、`serial` は明示時のみ | breaking change として扱う | resolved |
| `serial` の意味は | 現在の逐次実行と同等を維持 | 既存フローを fallback として残す | resolved |
| 初回バッチサイズは | `min_reviews` 件 | 初回のレビュー待ち時間を削減する | resolved |
| 2回目以降のバッチサイズは | `judge_every` 件 | 収斂判定頻度を既存パラメータで制御する | resolved |
| `max_reviews` 手前で不足する場合 | 残件のみ実行し、その後 judge 1回 | 上限超過せず最終判定できる | resolved |
| review 失敗時の扱い | `serial`/`parallel` とも即 `exit 22` | 失敗ポリシーを統一する | resolved |
| parallel で1件失敗時 | 他 review を即 cancel、judge は実行しない | fail-fast を厳格化する | resolved |
| 失敗時に残すエラー情報 | 最初に検知した1件のみ報告 | ログ/エラー契約を単純化 | resolved |
| review artifact の保存条件 | 成功 review のみ保存、失敗/キャンセルは保存しない | artifact と実行結果を一致させる | resolved |
| review 連番の扱い | 成功分のみ連番（欠番なし） | deterministic な検査を容易にする | resolved |
| judge artifact の保存条件 | 成功 judge + JSON parse 成功時のみ保存 | 不正 JSON の混入を防ぐ | resolved |
| judge 非0/parse失敗時 | 即 `exit 22`、artifact は保存しない | 判定結果の信頼性を担保する | resolved |
| `agent.max_iterations` の扱い | `review` では使用しない（run only） | run/review の責務境界を明確化 | resolved |
| `agent.sleep_seconds` の扱い | `review` では無効（待機なし） | review の総時間を短縮 | resolved |
| dry-run 表示 | review dry-run に `execution_mode` と batch 規則を表示 | 実行計画の可視性を高める | resolved |
| init 出力 | `execution_mode: parallel` を明示生成 | 暗黙 default 依存を避ける | resolved |
| PR の分割方針 | 本件は PR #1、CLI 導入は PR #2 | 設計/実装/レビュー境界が明確になる | resolved |

## Design

### 1. Config Model

`completion.review.review_convergence` に以下を追加する。

```yaml
completion:
  review:
    strategy: review_convergence
    review_convergence:
      execution_mode: parallel # parallel | serial
      min_reviews: 3
      max_reviews: 10
      judge_every: 2
      stable_rounds: 2
```

制約:

- 未指定は `parallel` を補完する。
- `parallel` / `serial` 以外は validation error（exit `22`）。
- 既存制約（`min_reviews >= 1`, `max_reviews >= min_reviews`, `judge_every >= 1`, `stable_rounds >= 1`）は維持する。

### 2. Review Scheduler Semantics

#### 2.1 `serial`

- 現行同等で 1 回に 1 role を実行する。
- `review` 非0終了は即 `exit 22`（この時点で artifact なし、count 加算なし）。
- `judge` 非0終了または judge JSON parse 失敗も即 `exit 22`。

#### 2.2 `parallel`

1. `review_count < min_reviews` の間は、`min(min_reviews - review_count, max_reviews - review_count)` 件を並列実行。
2. 以降は `min(judge_every, max_reviews - review_count)` 件を並列実行。
3. 各 review バッチ完了後、失敗がなければ `judge` を 1 回直列実行。
4. `review_count == max_reviews` 到達後は review を追加せず、未 judge の成功 review がある場合のみ最終 judge を 1 回実行。

### 3. Failure, Cancellation, and Completion

- review バッチ内で 1 件でも非0終了を検知した時点で batch context を cancel する。
- cancel 後は worker を join して終了状態を回収し、成功 review のみ保存したうえで `exit 22` を返す。
- judge は「バッチ内 review 全成功」のときのみ実行する。
- judge 0終了でも JSON parse 失敗時は `exit 22`。
- 収斂判定は既存どおり `signal == completion.review.signal && new_findings == 0` を judge 成功時のみ評価する。

### 4. Artifact and State Rules

- `REVIEW_XXXX.md`: 成功した review のみ保存。
- `JUDGE_XXXX.json`: 成功した judge かつ parse 成功時のみ保存。
- 失敗/キャンセル review、失敗/parse失敗 judge は保存しない。
- `review_count` / `reviews_since_judge` / `judge_count` は成功時のみ加算。
- 採番は成功分のみ連番（欠番なし）とする。
- parallel の review 起動順は deterministic（常に 1..N）とする。

### 5. Run/Review Control Separation

- `run`: 従来どおり `agent.max_iterations` / `agent.sleep_seconds` を使用。
- `review`: 上記 2 パラメータを使用しない。
- `review` の終了条件は以下のみ。
  - converged: exit `0`
  - `max_reviews` 到達で未収束: exit `23`
  - runtime error（review/judge non-zero, parse failure 等）: exit `22`

### 6. Dry-run and Init

- `ralph review --dry-run` に以下を追加表示。
  - `execution_mode`
  - parallel batch 規則（initial=`min_reviews`, steady=`judge_every`）
  - `agent.max_iterations` / `agent.sleep_seconds` が run only である旨
- `ralph init` の config テンプレートに `execution_mode: parallel` を明示出力する。

### 7. Testing Strategy

- config load/validate: `execution_mode` default 補完と enum reject。
- scheduler: initial/steady/partial batch と `max_reviews` 境界。
- runtime failure: review fail-fast（serial/parallel）、parallel cancel、judge non-zero、judge parse failure。
- artifact/state: 成功時のみ保存/加算、連番欠番なし。
- dry-run/init: 表示・生成内容の回帰固定。

## Compatibility & Sunset

この変更は breaking change を含む。

- `execution_mode` 未指定時の挙動が implicit serial から implicit parallel に変わる。
- `review` で `agent.max_iterations` / `agent.sleep_seconds` が効かなくなる。

### Temporary Mechanism Ledger

| ID | Mechanism | Introduced For | Retirement Trigger | Retirement Verification | Owner | Target |
|----|-----------|----------------|--------------------|-------------------------|-------|--------|
| TEMP01 | 旧 default（implicit serial）前提の運用手順 | 既存利用者の運用慣性を段階的に切替えるため | `execution_mode: parallel` が init 既定として浸透し、旧前提を参照する手順書を削除 | init 生成物テストで `execution_mode: parallel` を固定し、README/設計文書に serial default 記述が残っていないことを確認 | ralph maintainers | PR #1 merge |
| TEMP02 | review が run ループ制御を共有する現行実装 | run/review 責務分離までの暫定構造 | review 専用ループへ分離し、`max_iterations/sleep_seconds` 依存を除去 | runner テストで review が `max_reviews` のみを上限として終了し、sleep 呼び出しが発生しないことを確認 | ralph maintainers | PR #1 merge |

## Decision Log

| ADR | Decision | Status |
|-----|----------|--------|
| [0004](../adr/0004-review-convergence-mode.md) | `ralph review` と review/judge 2-role 収斂モデルの導入 | Proposed |
| [0006](../adr/0006-review-parallel-default-and-loop-separation.md) | review を parallel default に変更し、run/review ループ制御責務を分離する | Proposed |

## Open Questions

なし

## Acceptance Criteria

1. `completion.review.review_convergence.execution_mode` が `parallel|serial` を受理し、未指定時は `parallel` になる。
2. `execution_mode` が不正値の場合は validation error（exit `22`）になる。
3. `parallel` は初回 `min_reviews` 件を並列実行し、以降は `judge_every` 件を並列実行する。
4. `max_reviews` 手前で残件が `judge_every` 未満の場合、残件のみ review 実行後に judge を実行する。
5. `serial`/`parallel` ともに review 非0終了で即 `exit 22` になる。
6. `parallel` で review 1件失敗時は他 review を cancel し、judge を実行しない。
7. `serial`/`parallel` ともに judge 非0終了で即 `exit 22` になる。
8. judge 0終了でも JSON parse 失敗時は `exit 22` になり、judge artifact は保存されない。
9. review artifact は成功 review のみ保存され、失敗/キャンセル review は保存されない。
10. judge artifact は成功 judge かつ parse 成功時のみ保存される。
11. review/judge の採番は成功分のみ連番（欠番なし）になる。
12. review state の count は成功時のみ加算される。
13. `review` モードは `agent.max_iterations` を使わず、未収束時は `max_reviews` 到達で `exit 23` になる。
14. `review` モードは `agent.sleep_seconds` を使わず、待機なしで進行する。
15. `ralph review --dry-run` は `execution_mode` と parallel batch 規則、run-only 設定注記を表示する。
16. `ralph init` 生成 config に `execution_mode: parallel` が明示出力される。
17. 本 PR には CLI ライブラリ導入差分が含まれない。
