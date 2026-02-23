# Config Schema Sync Implementation Plan

**Source**: `docs/plans/2026-02-24-config-schema-sync-design.md`
**Trace Pack**: `docs/plans/2026-02-24-config-schema-sync-plan.trace.md`
**Compose Pack**: `docs/plans/2026-02-24-config-schema-sync-plan.compose.md`
**Goal**: `internal/config` 実装と `schemas/config.schema.json` のドリフトを自動生成と CI 検証で防止する。
**Architecture**: `internal/config` 型定義を schema 生成の入力にし、`jsonschema-go` + 最小ポスト処理で schema を生成する。runtime の意味制約は既存 `Validate` を維持し、Task/CI にドリフト検知を組み込む。
**Tech Stack**: Go, `github.com/google/jsonschema-go`, Task, YAML/JSON schema validation.

## Task Dependency Graph

`T1 -> T2 -> T3 -> T4`

### Task 1: Build schema generation foundation

**Satisfied Requirements**: REQ01, REQ02, REQ04, AC01  
**Design Anchors**: GOAL01, GOAL04, DEC01, DEC02  
**Goal**: `internal/config` 型から `schemas/config.schema.json` を生成する基盤を導入し、schema artifact をリポジトリ管理に戻す。  
**Dependencies**: none

**Files:**
- Create: `internal/config/schema/generator.go` (schema 生成ロジック)
- Create: `internal/config/schema/generator_test.go` (生成結果の契約テスト)
- Create: `schemas/config.schema.json` (generated artifact)
- Modify: `internal/config/types.go` (schema 生成用 `json` タグ整備)
- Modify: `go.mod` (schema 生成依存の追加)

**RED**
- schema ファイル未生成時または key 名が runtime 仕様と不一致の場合に失敗する生成テストを追加する。
- 生成ファイルに generated ヘッダーがない場合に失敗するテストを追加する。
- Run: `go test ./internal/config/schema -run TestGenerate`
- Expected: `FAIL with assertion mismatch for missing artifact/header/field mapping`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- `jsonschema-go` を使った生成処理を実装し、`schemas/config.schema.json` を出力する。
- `internal/config` の `yaml` タグと整合する `json` タグを追加し、optional 項目の required 推論が設計どおりになるよう整理する。

**REFACTOR**
- 生成オプションと出力整形（indent/改行/ヘッダー）をヘルパー化して、再利用可能な API に整理する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `schemas/config.schema.json` が存在し、`internal/init/templates/config.tmpl` の `$schema` URL と整合する（REQ01, AC01）。
- 生成は `internal/config` 型定義を入力として再現可能である（REQ02）。
- generated ヘッダーが常に付与され、手編集抑止の契約がテストで担保される（REQ04）。
- Run: `go test ./internal/config/schema ./internal/init -run 'TestGenerate|TestScaffold'`
- Expected: `PASS`

### Task 2: Encode runtime-equivalent schema constraints

**Satisfied Requirements**: REQ03, REQ07, AC04, AC05  
**Design Anchors**: GOAL03, GOAL04, DEC01, DEC02, NONGOAL01, NONGOAL02  
**Goal**: schema で表現可能な runtime 制約を post-process で反映し、unknown fields 非許容の二重防御を維持する。  
**Dependencies**: T1

**Files:**
- Modify: `internal/config/schema/generator.go` (post-process 制約の付与)
- Modify: `internal/config/schema/generator_test.go` (const/enum/XOR/additionalProperties 契約テスト)
- Modify: `internal/config/load_test.go` (runtime unknown fields 非許容の回帰テスト維持/拡張)
- Modify: `schemas/config.schema.json` (制約反映後の generated artifact)

**RED**
- `version const`, `git.commit enum`, `step.on_fail enum`, `step.uses enum`, `run/uses xor(oneOf)` が schema にない場合に失敗するテストを追加する。
- `additionalProperties: false` が欠落した場合に失敗するテストを追加する。
- Run: `go test ./internal/config/schema -run TestSchemaConstraints`
- Expected: `FAIL with assertion mismatch for missing constraint keywords`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- 生成後 schema に runtime 等価の制約を付与し、設計で指定された構造制約を反映する。
- runtime `KnownFields(true)` を維持し、schema 側 `additionalProperties` との整合を確保する。

**REFACTOR**
- post-process ルールをデータ駆動化し、今後の制約追加時にテストと一緒に拡張しやすくする。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- generated schema が `version`, `git.commit`, `step.on_fail`, `step.uses`, `run/uses XOR` を表現する（REQ03, AC04）。
- unknown fields 非許容が runtime (`KnownFields(true)`) と schema (`additionalProperties: false`) の双方で維持される（REQ07, AC05）。
- `Validate` の runtime 意味制約を廃止せず、schema-only 置換を行わない（NONGOAL01 guard）。
- `config.version` の型厳密化を新規導入しない（NONGOAL02 guard）。
- Run: `go test ./internal/config/...`
- Expected: `PASS`

### Task 3: Wire regeneration and drift detection into Task/CI flow

**Satisfied Requirements**: REQ05, REQ06, AC02, AC03  
**Design Anchors**: GOAL01, GOAL02, DEC02  
**Goal**: 開発者と CI が同じ操作で schema を再生成し、ドリフトを検出できる運用導線を確立する。  
**Dependencies**: T2

**Files:**
- Create: `cmd/ralph/schema.go` (`task schema` から呼ばれる生成エントリポイント)
- Modify: `Taskfile.yml` (`task schema` の追加と `task check` へのドリフト検証統合)
- Modify: `internal/config/schema/generator_test.go` (idempotence/再生成テスト)

**RED**
- `task schema` が未定義、または再実行しても差分ゼロにならない場合に失敗する検証を追加する。
- `task check` が schema ドリフトを検知しない場合に失敗する検証を追加する。
- Run: `task schema && git diff --exit-code -- schemas/config.schema.json`
- Expected: `FAIL with non-zero diff when regeneration pipeline is incomplete`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- `task schema` を追加し、schema 生成エントリポイントを標準化する。
- `task check` に再生成＋差分検知を組み込み、CI で更新漏れをブロックする。

**REFACTOR**
- Taskfile 上の検証コマンドを変数化し、ローカル/CI の実行順序差異を最小化する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `task schema` 実行で `schemas/config.schema.json` が再現可能に生成される（REQ05, AC02）。
- `task check`（または同等 CI）で schema ドリフトが検出される（REQ06, AC03）。
- Run: `task schema && git diff --exit-code -- schemas/config.schema.json && task check`
- Expected: `PASS`

### Task 4: Close rollout documentation and guardrails

**Satisfied Requirements**: REQ01, REQ08, AC01  
**Design Anchors**: GOAL02, GOAL04, DEC02, NONGOAL03  
**Goal**: 生成運用の手順を canonical guidance に反映し、`.ralph/` への schema 展開禁止を運用面でも固定する。  
**Dependencies**: T3

**Files:**
- Modify: `AGENTS.md` (`task schema` とドリフト検証フローを開発コマンドとして明記)
- Modify: `docs/plans/2026-02-24-config-schema-sync-design.md` (Open Questions の解消結果と最終運用を反映)
- Modify: `internal/init/scaffold_test.go` (`.ralph/config.schema.json` 非生成の回帰ガード明確化)

**RED**
- 開発手順に schema 再生成フローが未記載の場合に失敗するドキュメント整合チェックを追加する。
- `.ralph/` 配下に schema を生成してしまう回帰ケースを失敗として固定する。
- Run: `go test ./internal/init -run TestScaffold`
- Expected: `FAIL with assertion when `.ralph` schema expansion or documentation contract is missing`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- canonical guidance と設計書を更新し、schema 再生成手順・drift チェック・非展開方針を明示する。
- 回帰テストを更新し、`.ralph/` に schema を展開しない方針を維持する。

**REFACTOR**
- テスト名と失敗メッセージを運用方針に合わせて明確化し、将来の設計変更時に差分理由を追跡しやすくする。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- schema artifact と template URL の整合前提がドキュメント上で明示される（REQ01, AC01）。
- 開発者向け手順に schema 再生成と drift 検知の標準フローが記載される（REQ08）。
- `.ralph/` 配下に schema 実ファイルを展開しない方針がテストで維持される（NONGOAL03 guard）。
- Run: `go test ./internal/init -run TestScaffold && task check`
- Expected: `PASS`

## Checkpoint Summary

- Alignment Verdict: PASS
- Forward Fidelity: PASS
- Reverse Fidelity: PASS
- Non-Goal Guard: PASS
- Granularity Guard: PASS
- Trace Pack: `docs/plans/2026-02-24-config-schema-sync-plan.trace.md`
- Compose Pack: `docs/plans/2026-02-24-config-schema-sync-plan.compose.md`
- Updated At: `2026-02-24`
