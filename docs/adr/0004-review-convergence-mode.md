# 0004: Add Review Convergence Mode with Structured Judge Output

## Status

Proposed

## Context

現行の `ralph run` は `tail_match` 完了判定のみを前提としており、「複数回レビューして収斂したら停止する」運用を直接表現できない。

要件としては以下が確定している。

- `ralph run` と `ralph review` を分離し、同一 `.ralph/config.yml` で使い分ける
- review ループは `review` role と `judge` role の 2 役で構成する
- 収斂制御は `min_reviews` / `max_reviews` / `judge_every` / `stable_rounds` で設定可能にする
- `run_id` は timestamp 形式に統一する
- `new_finding_keys` は v1 で必須にする
- 非収斂時の上限到達は既存 exit code `23` を再利用する

また、単なる signal 一致のみでは判定の曖昧さが残るため、judge 出力は機械可読な JSON 契約に固定する必要がある。

## Decision

`ralph review` モードを追加し、以下を採用する。

- command 別 completion profile:
  - `completion.run.strategy = tail_match`
  - `completion.review.strategy = review_convergence`
- `agent.command` は廃止し、`agent.run_command` を必須化する
  - `agent.review_command` / `agent.judge_command` は任意 override
- judge は `.ralph/reviews/<run_id>/JUDGE_XXXX.json` を出力する
  - `run_id` は UTC timestamp 形式 `YYYYMMDDTHHMMSSZ`
- judge JSON の必須キーは `signal`, `new_findings`, `new_finding_keys`
  - `new_finding_keys` は v1 必須
- 安定判定は `signal == completion.review.signal` かつ `new_findings == 0`
- `stable_rounds` 連続で安定判定成立、かつ `min_reviews` 以上で完了（exit `0`）
- `max_reviews` 到達時に未収斂なら exit `23`
- `phases.pre` / `phases.post` は `run` 専用とし、`review` では実行しない
- review scope 解決は `prd.plan` が指す plan.md を SoT とし、`stories[].passes/deps` をレビュー対象選択に使わない
- `review` / `judge` の stderr ログは v1 では自由形式とし、固定キー契約は設けない
- `ralph init` は prompt を 3 分割で生成する（`prompt.run.md`, `prompt.review.md`, `prompt.judge.md`）
  - `prompt.md` は生成しない
- `ralph init` の `config.yml` は最小構成として `agent.run_command` と `completion.run` / `completion.review` を含む
  - `review_command` / `judge_command` は初期出力しない
- `ralph init` は `.ralph/reviews/` を生成しない（`ralph review` 実行時に必要なら作成）

## Consequences

### Positive

- run/review を同一 config で明確に切り替えられる。
- judge 判定が JSON 契約により再現可能になり、自動判定しやすい。
- `new_finding_keys` 必須化により差分追跡の安定性が上がる。
- init 直後から run/review の入力ファイル構成が明確になる。

### Negative

- config 仕様に breaking change が入る（`agent.command` 廃止、completion の profile 化）。
- `ralph review` 実装に追加の validation と状態管理が必要になる。
- judge JSON 契約を守らない command では runtime error（exit `22`）が増える可能性がある。
- prompt ファイル数が増え、初期学習コストがわずかに上がる。

### Neutral

- `tail_match` の runtime 挙動そのものは維持される。
- 終了コード体系は拡張せず、既存の `23` を再利用する。
