# 0006: Embed Bun-compiled Agent Gateway for Main Execution

## Status

Proposed

## Date

2026-02-28

## Deciders

- @shuymn

## Context and Problem Statement

`ralph` の main 実行は現在 `sh -c <agent command>` で外部 CLI を直接起動している。  
この方式では provider（Codex/Claude）差分が runner 側へ漏れやすく、SDK 変更追従の影響範囲が広い。

一方で、loop 制御・completion 判定・git 制御は Go 実装で安定しており、フル TypeScript 移行は再実装コストと回帰リスクが高い。  
また、運用要件として「gateway の手動起動不要」「Node 依存追加なし」「v1 から structured output 契約を強制」が求められる。

## Decision Drivers

- 既存 Go control plane（loop/config/completion/git）を維持すること。
- ユーザー操作を `ralph run` / `ralph review` のまま保つこと。
- SDK 差分の影響範囲を最小化すること。
- Node ランタイム依存なしで配布すること。
- `run/review/judge` 全 role で structured output 契約を維持すること。

## Considered Options

1. 現行の shell command 実行を継続する。
2. フル TypeScript 移行を行う。
3. Go runner + TypeScript gateway のハイブリッド構成へ移行する。

## Decision Outcome

採用: **Option 3（Go runner + TypeScript gateway）**

- Go は control plane（loop/config/completion/git/exit code）を担当する。
- TS gateway は execution plane（SDK 呼び出し、stream 正規化、`run_turn` 応答）を担当する。
- protocol contract は Go/TS の型定義 + 共有 fixture を SoT とし、v1 では standalone jsonschema artifact を管理しない。
- transport は v1 で `stdio + JSON-RPC`、`stream_event` は `stderr` 固定とする。
- `run_turn.output_schema` と `final_result.structured_output` は全 role で必須とする。
- gateway は bun native compile（`darwin/arm64`）を Go バイナリに embed し、実行時に自動起動する。
- `agent.run_command` / `agent.review_command` / `agent.judge_command` は廃止する。

採用理由（なぜ勝つか）:

- Option 1（shell command 継続）は SDK 差分が runner 側へ漏れ続け、structured output 契約を role 全体で強制しづらいため棄却。
- Option 2（フル TypeScript 移行）は Go の loop/completion/git を再実装するコストと回帰リスクが高く、現行資産の再利用効率が低いため棄却。

境界ルール:

- Go は provider SDK を直接呼び出さない。
- TS は loop/completion/git/exit code 判定を持たない。

## Consequences

### Positive

- SDK 差分を TS gateway に隔離できる。
- 既存 Go runner 資産を維持したまま拡張できる。
- 手動起動不要の UX を維持できる。
- bun compile + embed により Node 依存なしで配布できる。

### Negative

- JSON-RPC 契約とプロセス管理の実装複雑性が増える。
- role ごとの output schema 設計・維持コストが増える。
- Go/TS 双方の型定義と共有 fixture を同期維持するコストが増える。
- バイナリサイズが増加する。
- 初期対応プラットフォームが `darwin/arm64` に限定される。
- 既存 command ベース config との互換性は失われる。

### Neutral

- exit code 体系（`20/21/22/23`）は維持する。
- `completion.run` / `completion.review` の判定仕様は維持する。
- `review` artifact（`REVIEW_n.md`, `JUDGE_n.json`）の保存規約は維持する。

## Validation

- Design Doc の `Acceptance Criteria` を満たすこと。
- protocol 契約は Go/TS 双方で共有 fixture（valid/invalid message）を用いて検証すること。
- `run/review/judge` の structured output 必須要件がテストで担保されること。
- `darwin/arm64` 非対応環境で明示エラーを返すこと。

## Sunset Clause

| TEMP ID | Retirement Trigger | Retirement Verification | Removal Scope |
|---------|--------------------|-------------------------|---------------|
| TEMP01 | `linux/amd64`, `linux/arm64`, `windows/amd64` の gateway 配布と CI 成功 | 対象 OS/arch で `ralph run --dry-run` と e2e が green | `darwin/arm64` 限定分岐と単一 bundle 前提の packaging/起動ロジックを削除し、multi-platform bundle 解決へ置換する |
| TEMP02 | エラー分類（retryable/non-retryable）を protocol に導入 | retry policy table の導入と分類別テスト追加 | 「固定1回のみ再試行」の実装分岐を削除し、分類駆動 retry policy に置換する |

## Links

- Design: [docs/plans/2026-02-28-go-ts-agent-gateway-design.md](../plans/2026-02-28-go-ts-agent-gateway-design.md)
- Related ADR: [0002-generate-config-schema-from-go-types.md](./0002-generate-config-schema-from-go-types.md)
