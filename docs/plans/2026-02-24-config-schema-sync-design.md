# Config Schema Drift Prevention - Design

## Overview

`ralph init` が出力する `.ralph/config.yml` は `schemas/config.schema.json` を参照しているが、現状は schema 実体が未配備である。さらに、`internal/config` の実装変更時に schema 更新漏れが起きると、エディタ補完・検証と実ランタイム挙動が乖離する。

本設計は、`internal/config` の Go 型を起点に `schemas/config.schema.json` を自動生成し、CI でドリフトを検知することで、schema と実装の同期を継続的に保証する。

## Goals

- `schemas/config.schema.json` を実装由来で生成し、手動更新を不要にする。
- schema と runtime 実装のドリフトを CI で検出してマージ前に防止する。
- 既存方針である unknown fields 非許容（`KnownFields(true)`）を維持する。
- 既存ランタイム仕様（defaults, `if` パース, `on_fail` 制約など）を破壊せず導入する。

## Non-Goals

- `internal/config.Validate` を JSON Schema だけで完全置換すること。
- `config.version` の YAML 型厳密性（`1` vs `"1"`）を強制すること。
- `.ralph/` 配下に schema 実ファイルを展開すること。

## Background

- 現行コード:
  - `.ralph/config.yml` テンプレートは remote schema URL を参照している。
  - `LoadBytes` は `yaml.Decoder.KnownFields(true)` により unknown fields を拒否している。
  - 仕様の多くは `internal/config.Validate` に実装されている。
- 課題:
  - `schemas/config.schema.json` が欠落しており、参照先が 404 相当になる。
  - 将来 schema を追加しても、手作業更新だと実装変更とのズレが再発する。

## Design

### 1. Source of Truth の分離

- 構造契約（項目名・型・required・additionalProperties）は生成 schema で表現する。
- 振る舞い契約（`if` の構文解析、`run/uses` 実行時意味、defaults 適用順、重複 step 名チェックなど）は `Validate` を SoT として維持する。
- これにより schema は「編集時の静的検証」と「公開契約」を担当し、実行時妥当性は従来どおり Go コードで担保する。

### 2. 生成方式

- ライブラリ: `github.com/google/jsonschema-go` を採用する。
- 生成入力:
  - `internal/config` の型定義に `json` タグを追加し、`yaml` タグと同じキー名を付与する。
  - optional 項目には `json:",omitempty"` を付与し、required 推論を runtime 仕様に合わせる。
- 生成後ポスト処理:
  - JSON Schema で表現可能かつ runtime と同値な制約のみを追加する。
  - 例:
    - `version` の const (`"1"`)
    - `git.commit` enum (`split`, `together`)
    - `step.on_fail` enum (`continue`, `stop_loop`)
    - `step.uses` enum (`auto_commit`)
    - `step.run` と `step.uses` の XOR (`oneOf`)
- 出力先:
  - `schemas/config.schema.json`（リポジトリ管理物）

### 3. ツールチェーン統合

- `task schema` を追加し、schema 生成を標準化する。
- `task check` に schema ドリフト検証を追加する。
  - 想定: `task schema` 実行後に `git diff --exit-code -- schemas/config.schema.json`
- 生成ファイル先頭に「generated」ヘッダーを付与し、手編集を抑止する。

### 4. 既存仕様との整合

- unknown fields 非許容は維持する（schema の `additionalProperties: false` と `KnownFields(true)` の二重防御）。
- `version` の型厳密性は要求しない（値制約は `"1"` を維持しつつ、YAML デコード挙動に依存する）。
- `ralph init` の schema 参照 URL は維持し、参照先実体として本生成物を公開する。

### 5. ロールアウト

1. 生成基盤の導入（generator + `task schema` + 初回 schema 生成）
2. CI ドリフト検証の導入（`task check` への統合）
3. ドキュメント更新（開発フローに「型変更時は schema 再生成」を明記）

## Alternatives Considered

- 手書き schema を維持する:
  - 却下。更新漏れを構造的に防げない。
- schema 用の別 struct を新設して生成する:
  - 却下。runtime struct との二重管理が発生し、別のドリフト源になる。

## Decision Log

| ADR | Decision | Status |
|-----|----------|--------|
| [0001](../adr/0001-go-runner-over-shell-generation.md) | Go runner が `.ralph/config.yml` を直接実行する | Accepted |
| [0002](../adr/0002-generate-config-schema-from-go-types.md) | `jsonschema-go` で `schemas/config.schema.json` を実装から生成する | Proposed |

## Open Questions

- [ ] 生成コマンドの配置を `cmd/`（専用 CLI）と `internal/`（テスト専用呼び出し）どちらに置くか。
- [ ] schema の整形規則（key 順序、indent、末尾改行）をどこまで固定するか。
- [ ] 生成ポスト処理を Go コードで実装するか、JSON Patch で宣言的に実装するか。

## Acceptance Criteria

1. `schemas/config.schema.json` がリポジトリに存在し、`internal/init/templates/config.tmpl` の `$schema` URL と整合している。
2. `task schema` 実行で `schemas/config.schema.json` を再現可能に生成できる。
3. `task check`（または同等 CI）で schema ドリフトが検出される。
4. generated schema は少なくとも `version`, `git.commit`, `step.on_fail`, `step.uses`, `run/uses XOR` を runtime 仕様どおり表現する。
5. unknown fields 非許容の仕様が維持される（runtime + schema の双方で確認できる）。
