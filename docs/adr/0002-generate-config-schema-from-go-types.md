# 0002: Generate Config Schema from Go Types

## Status

Proposed

## Context

`.ralph/config.yml` は remote URL の JSON Schema を参照する設計だが、`schemas/config.schema.json` が欠落している。さらに、schema を手作業で管理すると `internal/config` の変更と同期が崩れ、エディタ検証結果と実行時挙動が乖離するリスクが高い。

一方で runtime 妥当性には `Validate` の独自ルール（`if` 式 parse、`run/uses` 意味制約、defaults など）があり、schema だけで完全代替はできない。

## Decision

`github.com/google/jsonschema-go` を採用し、`internal/config` の Go 型から `schemas/config.schema.json` を自動生成する。

- 生成 schema は構造契約（型、required、additionalProperties）を担う。
- runtime の意味制約は `internal/config.Validate` を維持する。
- schema 生成後に runtime と等価な最小制約（`version` const、enum、`run/uses` XOR）をポスト処理で付与する。
- CI で再生成差分をチェックし、schema ドリフトをブロックする。

## Consequences

### Positive

- schema 欠落問題を解消できる。
- 実装変更時の schema 更新漏れを CI で検知できる。
- unknown fields 非許容方針を schema/runtime 双方で整合させられる。

### Negative

- 生成処理とポスト処理の保守コストが増える。
- `json` タグ付与や required 制御（`omitempty`）の設計を誤ると schema が過剰/過少制約になる。

### Neutral

- `ralph init` が `.ralph/` 配下へ schema 実ファイルを展開しない方針は維持される。
- `config.version` の型厳密性を高める方針は採用しない（現状どおり）。
