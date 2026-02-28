# 0006: Embed Bun-compiled Agent Gateway for Main Execution

## Status

Proposed

## Context

`ralph` の main 実行は現在 `sh -c <agent command>` で外部 CLI を直接起動している。  
この方式では provider（Codex/Claude）差分が runner 側へ漏れやすく、SDK 変更追従の影響範囲が広い。

一方で loop 制御、completion 判定、git 制御は Go 実装で安定しており、フル TypeScript 移行は再実装コストと回帰リスクが高い。  
また、運用要件としてユーザーに gateway 手動起動を要求しないこと、Node 依存を追加しないことが求められる。

## Decision

main 実行経路を Go runner + TypeScript gateway のハイブリッド構成へ移行する。

- Go は loop / config validation / completion / git 制御を継続担当する。
- TypeScript gateway は `codex-sdk` / `claude-agent-sdk` 呼び出しを担当する。
- Go と gateway の通信は v1 で `stdio + JSON-RPC` を採用する（Unix socket は非対応）。
- protocol contract は `docs/protocol/agent-gateway-v1.json` を SoT として固定する。
- gateway は bun native compile で `darwin/arm64` 実行ファイル化し、Go バイナリへ embed する。
- Go が実行時に展開・checksum 検証・自動起動する。
- `agent.run_command` / `agent.review_command` / `agent.judge_command` は廃止し、role 別 provider 設定に置き換える。
- `run` / `review` / `judge` ごとに provider を設定可能とし、`review` / `judge` 未指定時は `run` を fallback とする。
- Claude provider の `setting_sources` 既定値は `project,user,local` とする。
- `stream_event` 出力は v1 で `stderr` 固定とし、CLI での出力先切替は提供しない。

## Consequences

### Positive

- SDK 差分を TS gateway に隔離できる。
- 既存 Go runner 資産を維持したまま拡張できる。
- ユーザー体験を `ralph run` / `ralph review` のまま維持できる（手動起動不要）。
- bun compile + embed により Node 依存なしで配布できる。

### Negative

- JSON-RPC protocol とプロセス管理の実装複雑性が増える。
- バイナリサイズが増加する。
- 初期対応プラットフォームが `darwin/arm64` に限定される。
- 既存 command ベース config との互換性は失われる。

### Neutral

- exit code 体系（`20/21/22/23`）は維持する。
- `completion.run` / `completion.review` の判定仕様は維持する。
- `review` artifact（`REVIEW_n.md`, `JUDGE_n.json`）の保存規約は維持する。
