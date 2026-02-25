# 0005: Remove Internal Config Compatibility Aliases After Migration

## Status

Accepted

## Context

`agent.command` 廃止と `completion.run` / `completion.review` 導入時に、内部移行を段階化するため以下の一時 alias を持っていた。

- `Agent.Command`
- `Completion.Strategy`
- `Completion.Signal`
- `Completion.TailLines`

これらは `json:"-" yaml:"-"` の in-memory bridge であり、外部 YAML 互換は提供していなかった。  
実際に loader は既に旧キーを reject しており、全 `.ralph/config.yml` の移行完了後は alias を残す理由がない。

## Decision

- 上記 compatibility alias を削除する。
- runner の command 解決は `agent.run_command` のみを SoT とする。
- run completion 判定は `completion.run` profile のみを参照し、旧トップレベル completion への fallback は廃止する。
- 旧キーは引き続き parse/validation error（exit `22`）として reject する。

## Consequences

### Positive

- config モデルの責務が単純化され、意図しない fallback 経路を排除できる。
- 旧形状の in-memory 依存が消え、今後の refactor とテスト保守が容易になる。

### Negative

- alias を前提にした内部テスト・補助コードは更新が必要になる。

### Neutral

- 外部 YAML 互換性の扱いは変わらない（旧キー reject は従来どおり）。
