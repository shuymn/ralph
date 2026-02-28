# Go + TS Agent Gateway Integration - Design

## Overview

本ドキュメントは、`ralph` の main 実行経路を `sh -c <agent command>` から、Go runner + TypeScript gateway（bun native compile 済みバイナリ）構成へ移行する設計を定義する。  
Go は既存どおり loop / config validation / completion / git 制御を担い、TS gateway は `codex` / `claude` SDK 呼び出しを担う。

本変更は breaking change とし、既存の `agent.run_command` / `agent.review_command` / `agent.judge_command` は廃止する。  
`run` / `review` / `judge` は role 別 provider 設定で切り替える。

## Goals

- `ralph run` / `ralph review` の両 mode で gateway 経由実行を統一する。
- TS 側 SDK 依存（Codex/Claude）を Go 本体から分離し、追従コストを局所化する。
- ユーザー体験として「手動で gateway を先に起動しない」運用を実現する（Go が自動起動）。
- 配布時に Node 依存を持たないため、bun native compile バイナリを `go:embed` して利用する。
- 既存 completion / review convergence / exit code 体系を維持する。

## Non-Goals

- v1 での Unix socket transport 対応（stdio のみ）。
- v1 での非 `darwin/arm64` ターゲット配布。
- 既存 command 設定との互換レイヤー維持。
- v1 での gateway 常駐デーモン管理コマンド（`start/stop/status`）追加。

## Background

現行 runner の main 実行は `sh -c <agent command>` で外部 CLI を直接起動する。  
この方式はシンプルだが、SDK 連携を進めると Go 側に provider 差分を持ち込みやすくなる。

一方で、フル TypeScript 移行は loop/validation/git の既存 Go 資産を再実装するコストが高い。  
そのため、Go 本体を維持しつつ TS gateway に SDK 呼び出しを委譲するハイブリッド構成を採用する。

## Clarifications

| Question | Answer / Assumption | Impact | Status |
|----------|----------------------|--------|--------|
| 対象 mode は `run` だけか？ | `run` / `review` の両方を対象とする。 | main 実行経路を mode 共通で gateway 化する。 | resolved |
| 既存 command 実行経路は残すか？ | 残さない。gateway を唯一経路とする breaking change。 | config 互換レイヤーを持たない。 | resolved |
| bun 配布ターゲットは？ | 初期は `darwin/arm64` のみ。 | 非対応プラットフォームは明示エラーで失敗。 | resolved |
| provider 対象は？ | 初期から `codex` / `claude` 両対応。 | TS gateway に provider adapter 2 系統を実装する。 | resolved |
| transport は？ | v1 は `stdio + JSON-RPC` のみ。 | Unix socket は将来拡張とする。 | resolved |
| provider 設定粒度は？ | `run` / `review` / `judge` で provider を個別指定可能。 | role-aware provider 解決ロジックが必要。 | resolved |
| role 別 provider 未指定時の挙動は？ | `review` / `judge` 未指定時は `run` provider へ fallback。 | 解決順を仕様化できる。 | resolved |
| gateway protocol を JSON schema として固定化するか？ | `docs/protocol/agent-gateway-v1.json` を SoT として固定化する。 | Go/TS の契約テストを schema ベースで運用できる。 | resolved |
| Claude settings/skills の読み込み方針は？ | `setting_sources` の既定値を `project,user,local` とする。 | provider ごとの設定差分を gateway 設定に吸収する。 | resolved |
| `stream_event` の既定出力先を切り替え可能にするか？ | v1 は CLI 切替オプションを追加せず、`stream_event` は `stderr` 固定とする。 | 機械可読出力（stdout）と進捗ログ（stderr）の分離を簡素に保証する。 | resolved |

## Design

### 1. Runtime Architecture

責務分離:

- Go (`ralph`):
  - config load/validation
  - loop scheduling（run/review/judge）
  - completion 判定
  - git 制御
  - gateway プロセス lifecycle 管理
- TS (`agent-gateway`):
  - provider adapter（codex-sdk / claude-agent-sdk）
  - turn 実行
  - stream event 送出
  - final result 返却

```mermaid
flowchart LR
  A["ralph run/review (Go)"] --> B["spawn embedded gateway (bun native binary)"]
  B --> C["stdio JSON-RPC initialize"]
  C --> D["run_turn(role, provider, prompt, context)"]
  D --> E["stream_event*"]
  D --> F["run_turn response(final_result)"]
  F --> G["Go completion/git/loop control"]
```

### 2. Gateway Packaging and Startup

- `agent-gateway` は bun native compile で `darwin/arm64` 実行バイナリを生成する。
- 生成物は圧縮アーカイブ + sha256 で Go バイナリに `go:embed` する。
- 実行時に `UserCacheDir` 配下へ展開し、checksum 検証後に実行する。
- Go は gateway 生存確認ではなく、`ralph` プロセス配下で子プロセスとして都度起動する（手動常駐不要）。
- 起動は `exec.CommandContext(context.Background(), <gateway>, "stdio")` を使い、終了時は graceful shutdown 後に kill fallback を持つ。

### 3. Transport and Protocol

v1 transport は stdio JSON-RPC のみ。

リクエスト:

- `initialize`
- `run_turn`
- `cancel_turn`
- `shutdown`

通知:

- `stream_event`（token/log/tool status など）

レスポンス:

- `run_turn` の返り値として `final_result` を返す（request/response 型）。

相関:

- JSON-RPC `id` で request/response を対応付ける。
- `stream_event` は `turn_id` と `seq` を必須にし、同一 turn 内で単調増加。
- `run_turn` ごとに `final_result` はちょうど 1 回。

### 4. Role-aware Provider Resolution

config で role 別 provider を指定する。

解決順:

1. `role_provider.<role>`
2. fallback `role_provider.run`

role:

- `run`
- `review`
- `judge`

provider 値:

- `codex`
- `claude`

### 5. Config Model (Draft)

```yaml
version: "1"

agent:
  gateway:
    transport: stdio
    startup_timeout_seconds: 10
    rpc_timeout_seconds: 1800
  role_provider:
    run: codex
    review: claude
    judge: claude
  providers:
    codex:
      # working_directory is inherited from runner working dir when omitted.
      working_directory: ""
    claude:
      # default: project,user,local
      setting_sources: ["project", "user", "local"]

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

breaking changes:

- remove: `agent.run_command`
- remove: `agent.review_command`
- remove: `agent.judge_command`

validation:

- `agent.gateway.transport` は `stdio` のみ許容。
- `role_provider.run` は必須。
- `role_provider.review` / `role_provider.judge` は任意（未指定時 fallback）。
- provider は `codex` / `claude` のみ。

### 6. Main Step Behavior

- `runMainStep` は shell command 実行を廃止し、gateway client 呼び出しへ置換する。
- 入力は従来どおり prompt file（judge は machine context prefix + prompt）を使用。
- gateway からの `stream_event` は runner の標準出力へ転送可能にする。
- `final_result.output` は tmpfile に保存し、既存 completion 判定ロジックを維持する。
- `review` mode の artifact 生成（`REVIEW_n.md` / `JUDGE_n.json`）は現行と同じ保存規約を維持する。

### 7. Failure Handling

- gateway 起動失敗、初期化失敗、RPC timeout、protocol violation は runtime error（exit `22`）。
- `run_turn` 実行中の gateway プロセス異常終了時は 1 回だけ再起動 + 同一 turn 再試行を許可する。
- 再試行後も失敗した場合は runtime error（exit `22`）。
- `cancel_turn` は SIGINT/SIGTERM 終了時の best-effort 経路として実装する。

### 8. UX

- ユーザー操作は `ralph run` / `ralph review` のまま変更しない。
- 手動で gateway を起動する必要はない。
- 初回起動は gateway 展開・起動により追加レイテンシが発生するが、loop 実行中は同一プロセスを再利用する。
- `stream_event` は v1 で `stderr` 固定出力とし、`stdout` は最終結果系出力に優先的に使う。

### 9. Build and Release

- build pipeline に gateway compile ステップを追加する。
- bun compile 出力をアーカイブ化し、sha256 を生成する。
- Go build 時に `darwin/arm64` 対象アーカイブを embed する。
- 非対応プラットフォームでは明示的に `errNoBundledGatewayForPlatform` を返し、回避方法を案内する。

### 10. Test Strategy

- config validation:
  - legacy command keys を拒否するテスト
  - role_provider fallback テスト
  - provider enum/transport enum テスト
- runner:
  - gateway mock/stub による run/review/judge の role 別実行テスト
  - stream_event と final_result 受信順/相関テスト
  - 再起動リトライ（1回）テスト
- packaging:
  - checksum 検証失敗テスト
  - materialize/cleanup テスト
  - unsupported platform テスト
- protocol:
  - `docs/protocol/agent-gateway-v1.json` と Go/TS 型定義の整合テスト
  - `run_turn`, `stream_event`, `final_result` の schema 検証テスト

### 11. Protocol Schema

- gateway contract の SoT は `docs/protocol/agent-gateway-v1.json` とする。
- JSON-RPC message の request/response/notification payload は同 schema で定義する。
- versioning は `protocol_version`（例: `v1`）で handshake し、互換性のない変更は `v2` として分離する。

## Compatibility & Sunset

この変更は breaking change であり、旧 config の後方互換は提供しない。

### Temporary Mechanism Ledger

| ID | Mechanism | Introduced For | Retirement Trigger | Retirement Verification | Owner | Target |
|----|-----------|----------------|--------------------|-------------------------|-------|--------|
| TEMP01 | `darwin/arm64` 限定の bundled gateway 配布 | 早期に gateway 統合を出荷するため | `linux/amd64`, `linux/arm64`, `windows/amd64` の gateway 配布と CI 成功 | 対象 OS/arch で `ralph run --dry-run` と e2e が green | @ralph-maintainers | 2026-Q2 |
| TEMP02 | gateway 異常終了時の「1回のみ」固定再試行 | 初期運用で過剰再試行を避けつつ可用性を確保 | エラー分類（retryable/non-retryable）を protocol に導入 | retry policy table の導入と分類別テスト追加 | @ralph-maintainers | 2026-Q2 |

## Decision Log

| ADR | Decision | Status |
|-----|----------|--------|
| [0006](../adr/0006-embed-bun-agent-gateway-for-main-execution.md) | Go runner + bun-compiled TS gateway の stdio JSON-RPC 構成を採用し、main 実行を gateway 専用化する | Proposed |

## Open Questions

- [ ] なし
