# Review Convergence Completion Strategy - Design

## Overview

本ドキュメントは、`ralph run` に加えて `ralph review` を導入し、「レビューを複数回実施し、収斂したら停止する」実行モードを追加する設計を定義する。  
既存の `tail_match` 完了戦略は維持しつつ、同一 config 内の command 別 profile（`completion.run` / `completion.review`）で並行運用可能にする。

本戦略では role を `review` と `judge` の 2 つに分離する。

- `review`: 毎回まっさらな前提でレビュー結果を `REVIEW_n.md` として出力する
- `judge`: 複数の `REVIEW_*.md` を統合し、`JUDGE_n.json` に構造化判定結果を出力する

CLI は `ralph run` と `ralph review` の 2 サブコマンドを提供する。  
どちらも同一の `.ralph/config.yml` を読み込む。

## Goals

- command 別 completion profile を導入し、`tail_match`（run）と `review_convergence`（review）を同一 config で共存できること。
- `review` / `judge` の 2 role 実行モデルを runner に導入できること。
- 収斂制御パラメータを config で指定できること（`min_reviews`, `max_reviews`, `judge_every`, `stable_rounds`）。
- 既存の終了コード体系を維持し、非収斂の上限到達は `23` を再利用できること。
- `ralph run`（通常ループ）と `ralph review`（収斂レビュー）を同一 config で切り替えて実行できること。

## Non-Goals

- 1 回の `ralph run` で `tail_match` と新戦略を同時評価するハイブリッド運用。
- 既存 pre/main/post モデルの廃止。
- runner がレビュー内容の意味理解を強制的に行う高度な自然言語解析。
- v1 での reviewer 並列実行（同時プロセス実行）。

## Background

現状の完了判定は `tail_match` のみであり、`stories[].passes` と signal 一致で停止する。  
このモデルでは「複数回レビューして、増分が止まったら終了」という反復収斂を直接表現できない。

一方で既存仕様との並行導入が要件であり、実行中に戦略を切り替えるハイブリッド運用は不要である。  
したがって、同一 config 内に command 別 profile（`completion.run` / `completion.review`）を持たせる設計を採用する。

## Clarifications

| Question | Answer / Assumption | Impact | Status |
|----------|----------------------|--------|--------|
| 既存要件との関係は？ | 既存要件とは並行導入。1 run 内でハイブリッド評価はしない。 | command ごとに単一 strategy を固定する。 | resolved |
| 新戦略の配置は？ | 同一 config の command 別 profile（`completion.run` / `completion.review`）で管理する。 | run/review を設定編集なしで併用しやすくなる。 | resolved |
| 実行モデルは？ | `review` と `judge` の 2 role。 | main 実行を role-aware にする必要がある。 | resolved |
| 収斂判定の頻度は？ | `judge_every` を設定可能（default 2）。 | role スケジューラが必要。 | resolved |
| 収斂安定条件は？ | `stable_rounds` を設定可能（default 2）。 | judge 連続安定回数の状態管理が必要。 | resolved |
| 最低レビュー回数は？ | `min_reviews` を設定可能（default 3）。 | 早期停止を防ぐ下限を設ける。 | resolved |
| 非収斂の安全弁は？ | `max_reviews` を設定可能（default 10）。 | review 実行回数ベースの停止条件が必要。 | resolved |
| `max_reviews` 到達時の終了コードは？ | 既存 `ExitCodeMaxIterations (23)` を再利用。 | 新規終了コードは不要。 | resolved |
| command は role ごとに分ける？ | ハイブリッド。`agent.run_command` を共通デフォルトにしつつ `review_command` / `judge_command` で上書き可。 | command 解決ルールを追加する。 | resolved |
| prompt はどう分離する？ | command/role 別ファイル（`.ralph/prompt.run.md`, `.ralph/prompt.review.md`, `.ralph/prompt.judge.md`）。 | 入力責務と検証対象が明確になる。 | resolved |
| reviewer は毎回まっさらか？ | yes（レビュー role は前回レビュー結果を入力として前提にしない）。 | role ごとの入力契約を明記する。 | resolved |
| `REVIEW_n.md` の配置場所は？ | `.ralph/reviews/<timestamp>/REVIEW_XXXX.md`（`timestamp`: UTC `YYYYMMDDTHHMMSSZ`）を既定とする。 | 過去 run との混在防止。 | resolved |
| CLI 形状は？ | `ralph run` と `ralph review` を分離し、両者とも同一 `.ralph/config.yml` を使う。 | command surface の追加と mode 固定が必要。 | resolved |
| judge の判定出力形式は？ | `JUDGE_n.json` を出力し、`ralph` が strict parse して判定に使う。 | signal 単独より再現性の高い収斂判定が可能。 | resolved |
| `new_finding_keys` は必須か？ | v1 で必須。 | 差分追跡の再現性を上げる。 | resolved |
| `review`/`judge` のログ形式は標準化するか？ | v1 は自由形式ログで運用し、固定項目は定義しない。 | 実装初期の拘束を減らし、運用結果を見て後続で標準化する。 | resolved |
| review scope は `stories[].passes/deps` で選ぶか？ | いいえ。review scope は `prd.plan` が指す plan.md を唯一の SoT とし、`stories` は参考情報のみ。 | 「eligible story なし」を誤検出するノイズを防ぎ、レビュー対象を実装コードに集中できる。 | resolved |
| `ralph init` の prompt 構成は？ | `.ralph/prompt.run.md` / `.ralph/prompt.review.md` / `.ralph/prompt.judge.md` を生成し、`.ralph/prompt.md` は生成しない。 | run/review の入力責務が明確になる。 | resolved |
| `ralph init` の config はどこまで生成する？ | `completion.run` と `completion.review` の両 profile を初期生成する。 | 生成直後から run/review を同一 config で使える。 | resolved |
| `ralph init` で `review_command` / `judge_command` は出力する？ | `config.tmpl` では省略し、必要時のみ利用者が追加する。 | 最小構成を維持しつつ拡張可能。 | resolved |
| `ralph init` で `.ralph/reviews/` を作るか？ | 作らない。`ralph review` 実行時に必要なら作成する。 | 初期スキャフォールドを最小化できる。 | resolved |

## Design

### 1. Command Profiles and Strategy Model

同一 `.ralph/config.yml` で command 別に completion profile を持つ。

- `completion.run.strategy = tail_match`（既存）
- `completion.review.strategy = review_convergence`（新規）

評価は command 内で排他的であり、1 回の command 実行で有効なのは 1 strategy のみとする。

### 1.1 Command Surface

- `ralph run`: 既存の通常ループ実行（main は `agent.run_command`、completion は既存設定を使用）
- `ralph review`: 収斂レビュー実行（`review`/`judge` をスケジュールし、completion は `review_convergence` を使用。`phases.pre` / `phases.post` は実行しない）

`ralph run` は `completion.run` を参照し、`ralph review` は `completion.review` を参照する。  
対応 profile が未定義、または strategy が不正な場合は設定エラー（exit `22`）で終了する。

### 2. Config Model (Draft)

```yaml
version: "1"

agent:
  run_command: "codex exec"
  max_iterations: 60
  sleep_seconds: 5

completion:
  run:
    strategy: tail_match
    signal: "<promise>COMPLETE</promise>"
    tail_lines: 20
  review:
    strategy: review_convergence
    signal: "<promise>COMPLETE</promise>"
    review_convergence:
      min_reviews: 3
      max_reviews: 10
      judge_every: 2
      stable_rounds: 2
```

制約:

- `run_command` は必須（非空）
- `review_command` / `judge_command` は任意（未指定時は `run_command` fallback）
- `completion.run.strategy == tail_match`
- `completion.review.strategy == review_convergence`
- `min_reviews >= 1`
- `max_reviews >= min_reviews`
- `judge_every >= 1`
- `stable_rounds >= 1`

judge JSON 契約（Draft）:

```json
{
  "signal": "<promise>COMPLETE</promise>",
  "new_findings": 0,
  "new_finding_keys": [],
  "total_findings": 12,
  "summary": "optional human summary"
}
```

制約:

- `signal`: string（judge の停止意思。設定値 `completion.review.signal` と完全一致で一致扱い）
- `new_findings`: integer（0 以上。前回 judge からの増分）
- `new_finding_keys`: string array（必須。重複排除・差分追跡補助）
- `total_findings`: integer（0 以上。任意の監視値）
- parse 不能・必須欠落は runtime error（exit `22`）

### 3. Role-specific Prompt Paths

command/role ごとに以下の固定 path を利用する。

- run: `.ralph/prompt.run.md`
- review role: `.ralph/prompt.review.md`
- judge role: `.ralph/prompt.judge.md`

必要な prompt が未存在・空ファイルの場合は設定エラー（exit `22`）とする。

### 4. Command Resolution

role ごとの実行 command は次の優先順で解決する。

1. `review` role: `agent.review_command`（非空なら採用）
2. `judge` role: `agent.judge_command`（非空なら採用）
3. fallback: `agent.run_command`

### 5. Runtime State

`review_convergence` 実行中は以下を runner state として保持する。

- `review_count`: 実行済み `review` 回数
- `reviews_since_judge`: 前回 judge 以降の `review` 回数
- `stable_count`: 連続で収斂と判定された judge 回数
- `judge_count`: 実行済み judge 回数（ログ用途）
- `run_id`: UTC timestamp 形式（`YYYYMMDDTHHMMSSZ`）

### 6. Scheduling Rule

role 選択は次のルールで決定する。

1. `review_count < min_reviews` の間は必ず `review`。
2. それ以外は、`reviews_since_judge >= judge_every` なら `judge`。
3. `review_count == max_reviews` の場合は追加レビューを禁止し、必要な judge を実行して判定する。

補足:

- `agent.max_iterations` は「総 role 実行回数」の安全弁としてそのまま有効。
- `max_reviews` は「review role 実行回数」の上限として別に有効。

### 7. Completion Rule (review_convergence)

judge role 実行結果（`JUDGE_n.json`）を parse して、次を満たすとき「この judge ラウンドが安定」と判定する。

- `signal == completion.review.signal`
- `new_findings == 0`

安定判定:

- 上記 2 条件を満たす: `stable_count++`
- いずれか不成立: `stable_count = 0`

停止条件:

- `review_count >= min_reviews` かつ `stable_count >= stable_rounds` のとき `0` で終了

非収斂終了:

- `review_count >= max_reviews` かつ停止条件未達の場合、`ExitCodeMaxIterations (23)` を返す

### 8. Review/Judge Artifacts

`review` role の成果物:

- `.ralph/reviews/<timestamp>/REVIEW_0001.md` のような連番ファイル

`judge` role の成果物:

- `.ralph/reviews/<timestamp>/JUDGE_0001.json` のような連番 JSON ファイル（必須）
- `.ralph/reviews/<timestamp>/JUDGE_0001.md` のような説明ファイル（任意）
- runner は JSON を parse し、`signal` と `new_findings` で安定判定を行う

`review` role の「まっさら」要件について:

- runner は `review` role への標準入力を `.ralph/prompt.review.md` のみに固定する
- review prompt は `prd.plan` が指す plan.md を scope の SoT とし、`stories[].passes/deps` による対象選択を行わない
- ただしワークスペース内ファイルの参照自体は command 側の振る舞いに依存するため、厳密制御は prompt 規約で担保する

### 9. Dry-run / Validation

`ralph run --dry-run` は `completion.run` profile を表示する。

- strategy (`tail_match`)
- `signal`, `tail_lines`
- `agent.run_command` の実行設定
- `prompt.run.md` path
- pre/post phase の step 実行計画（既存 dry-run と同等）

`ralph review --dry-run` は `completion.review` profile を表示する。

- strategy (`review_convergence`)
- role command の解決結果（review/judge, override/fallback 含む）
- `min_reviews`, `max_reviews`, `judge_every`, `stable_rounds`
- `prompt.review.md` / `prompt.judge.md` path
- phase は review mode で無効化されること
- judge JSON 契約（必須キーと判定条件）

validation 失敗時は既存同様 exit `22`。

### 10. Compatibility and Migration

- 本設計は config 仕様の breaking change を含む。
  - `agent.command` は廃止し、`agent.run_command` を必須とする。
  - completion 設定は command 別 profile（`completion.run` / `completion.review`）へ移行する。
  - 段階移行のために置いた internal compatibility alias は移行完了後に撤去する（[ADR 0005](../adr/0005-remove-config-compatibility-aliases.md)）。
- `review_convergence` は `ralph review` 実行時にのみ評価する。
- 終了コード体系は維持し、非収斂上限は `23` を再利用する。
- `ralph review` は新規導入だが、設定ファイルは分割せず `.ralph/config.yml` 1 枚を継続利用する。

### 11. `ralph init` Scaffolding

`ralph init` は以下を生成対象とする。

- `.ralph/config.yml`
- `.ralph/prompt.run.md`
- `.ralph/prompt.review.md`
- `.ralph/prompt.judge.md`
- `.ralph/prd.json`
- `.ralph/progress.md`

`ralph init` の仕様:

- 既存ファイルは上書きせず skip する（既存挙動を維持）。
- `config.yml` は最小構成として `agent.run_command` と `completion.run` / `completion.review` を含む。
- `review_command` / `judge_command` は初期出力しない（必要時に利用者が追記）。
- `.ralph/prompt.md` は生成しない。
- `.ralph/reviews/` は生成しない（`ralph review` 実行時に必要なら作成）。

## Alternatives Considered

- 既存 `tail_match` にレビュー収斂ロジックを直接混在させる:
  - 却下。ハイブリッド実行を招き、仕様分岐が曖昧になる。
- `ralph run` 単一コマンドに mode フラグを追加する:
  - 却下。CLI 利用時に意図しない mode を起動しやすく、運用上の誤実行リスクが高い。

## Decision Log

| ADR | Decision | Status |
|-----|----------|--------|
| [0001](../adr/0001-go-runner-over-shell-generation.md) | Go runner が config を直接実行する | Accepted |
| [0004](../adr/0004-review-convergence-mode.md) | command 別 completion profile と `ralph review` を導入し、judge JSON で収斂判定する | Proposed |
| [0005](../adr/0005-remove-config-compatibility-aliases.md) | 段階移行用の internal compatibility alias を撤去し、`run_command` / `completion.run` を唯一の SoT にする | Accepted |

## Open Questions

なし

## Acceptance Criteria

1. `completion.review.strategy=review_convergence` を設定すると、`ralph review` が review/judge role を自動スケジュールする。
2. `agent.review_command` / `agent.judge_command` が空の場合、`agent.run_command` が fallback として使われる。
3. `judge_every`（default 2）ごとに judge role が実行される。
4. `stable_rounds`（default 2）連続で `signal == completion.review.signal` かつ `new_findings == 0` が成立し、かつ `min_reviews`（default 3）以上のとき、exit `0` で終了する。
5. `review_count` が `max_reviews`（default 10）に達しても停止条件未達なら exit `23` で終了する。
6. `completion.review.strategy=review_convergence` かつ `.ralph/prompt.review.md` / `.ralph/prompt.judge.md` が欠落している場合は exit `22` で失敗する。
7. `completion.run.strategy=tail_match` の runtime 挙動（停止判定ロジック）は従来どおり維持される。
8. 旧キー（`agent.command`, 旧 completion 形式）を含む config は validation error（exit `22`）で失敗する。
9. `ralph run --dry-run` は `completion.run` profile（`tail_match`, `signal`, `tail_lines`, `agent.run_command`, `prompt.run.md`, pre/post step 計画）を表示する。
10. `ralph review --dry-run` は `completion.review` profile（`review_convergence` 設定、review/judge command 解決結果、review/judge prompt path、judge JSON 契約）を表示する。
11. `ralph review` は `completion.review.strategy=review_convergence` のときのみ実行可能で、それ以外は exit `22` で失敗する。
12. `ralph run` と `ralph review` は同一 `.ralph/config.yml` を参照し、設定ファイルの分割を要求しない。
13. `JUDGE_n.json` は `signal` / `new_findings` / `new_finding_keys` を必須とし、欠落または型不一致は exit `22` で失敗する。
14. `run_id` は UTC timestamp 形式（`YYYYMMDDTHHMMSSZ`）でディレクトリ名に使われる。
15. `review`/`judge` の stderr ログは v1 ではフォーマットを固定せず、特定キーの出力を必須化しない。
16. `ralph init` は `.ralph/config.yml`, `.ralph/prompt.run.md`, `.ralph/prompt.review.md`, `.ralph/prompt.judge.md`, `.ralph/prd.json`, `.ralph/progress.md` を生成する。
17. `ralph init` は `.ralph/prompt.md` と `.ralph/reviews/` を生成しない。
18. `ralph init` が生成する `config.yml` は `agent.run_command` と `completion.run` / `completion.review` を含み、`review_command` / `judge_command` は省略する。
19. `ralph run` 実行時に `.ralph/prompt.run.md` が欠落または空の場合は exit `22` で失敗する。
20. `ralph review` 実行時は `phases.pre` / `phases.post` を実行しない（phase は run 専用）。
