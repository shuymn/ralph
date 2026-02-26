# 0006: Make Review Parallel by Default and Separate Run/Review Loop Controls

## Status

Proposed

## Context

`ralph review` は現状 1 iteration = 1 role の直列実行であり、review 回数が増えるほど所要時間が増える。
また runner 実装は `run` と `review` のループ制御を共有しており、`review` でも `agent.max_iterations` / `agent.sleep_seconds` の影響を受ける。

運用要件として以下が確定した。

- review を最大限並列化し、judge は review 完了待ちの直列実行にする
- 並列/直列は config で切替可能にする
- default は `parallel`、`serial` は明示指定時のみ
- `review` / `judge` 非0終了は fail-fast（exit `22`）へ統一する
- `review` モードは `agent.max_iterations` / `agent.sleep_seconds` を使用しない

## Decision

以下を採用する。

- `completion.review.review_convergence.execution_mode` を追加する
  - 許可値: `parallel`, `serial`
  - 未指定時 default: `parallel`
- `parallel` のスケジューリングは以下とする
  - 初回: `min_reviews` 件の review を並列実行
  - 以降: `judge_every` 件の review を並列実行
  - 各バッチ後: judge を 1 回直列実行
  - `max_reviews` 手前では残件のみ review を実行し、その後 judge
- `serial` は既存逐次モデルを維持する（ただし失敗ポリシーは統一）
- `review` 非0終了時は `serial`/`parallel` とも即 `exit 22`
  - failed/canceled review artifact は保存しない
- `judge` 非0終了または judge JSON parse 失敗時は即 `exit 22`
  - failed/invalid judge artifact は保存しない
- artifact と count は成功時のみ更新する
  - `REVIEW_XXXX.md` / `JUDGE_XXXX.json` は成功分のみ連番（欠番なし）
- `review` モードでは `agent.max_iterations` / `agent.sleep_seconds` を無効化する
  - review の停止条件は converged / `max_reviews` 到達 / runtime error のみ

## Consequences

### Positive

- review の実行時間短縮が見込める。
- `run` と `review` の責務境界が明確になり、実装の見通しが改善する。
- 失敗時の挙動と artifact 契約が単純化され、デバッグしやすくなる。

### Negative

- `execution_mode` 未指定時の挙動が serial から parallel に変わるため、breaking change になる。
- parallel 実行で cancellation と結果集約の実装複雑度が上がる。
- テストケース（失敗系、境界系、artifact 整合）の拡充が必要になる。

### Neutral

- judge JSON 契約（`signal`, `new_findings`, `new_finding_keys`）自体は変更しない。
- CLI フラグは追加せず、切替は config で完結させる。
- CLI ライブラリ導入は別 PR（Design Doc #2）で扱う。
