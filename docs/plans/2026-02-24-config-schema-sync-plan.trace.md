# Config Schema Sync Plan Trace Pack

## Design Atom Index

### Goals

- GOAL01: `schemas/config.schema.json` を実装由来で生成し、手動更新を不要にする。
- GOAL02: schema と runtime 実装のドリフトを CI で検出してマージ前に防止する。
- GOAL03: unknown fields 非許容（`KnownFields(true)`）を維持する。
- GOAL04: defaults/`if`/`on_fail` を含む既存ランタイム仕様を壊さず導入する。

### Non-Goals

- NONGOAL01: `internal/config.Validate` を JSON Schema だけで完全置換しない。
- NONGOAL02: `config.version` の YAML 型厳密性を強制しない。
- NONGOAL03: `.ralph/` 配下に schema 実ファイルを展開しない。

### Decisions

- DEC01: Go runner の runtime 検証を正とし、構造契約のみを schema に担わせる（ADR-0001 を前提）。
- DEC02: `jsonschema-go` で schema を生成し、最小ポスト処理と CI ドリフト検知を組み合わせる。

### Requirements

- REQ01: `schemas/config.schema.json` をリポジトリ管理物として追加し、`config.tmpl` の `$schema` URL と整合させる。
- REQ02: `internal/config` 型を入力に schema 生成できるよう、`json` タグと optional 制御を整備する。
- REQ03: schema に `version const`, enum 群, `run/uses` XOR を反映する。
- REQ04: 生成物に generated ヘッダーを付与し、再生成で同一出力を得られるようにする。
- REQ05: `task schema` で schema 再生成を実行できるようにする。
- REQ06: `task check`（または同等 CI）に schema ドリフト検知を組み込む。
- REQ07: unknown fields 非許容を runtime と schema の双方で維持する。
- REQ08: schema 再生成運用を canonical guidance に反映する。

### Acceptance Criteria

- AC01: `schemas/config.schema.json` が存在し、`config.tmpl` の `$schema` URL と整合している。
- AC02: `task schema` 実行で `schemas/config.schema.json` を再現可能に生成できる。
- AC03: `task check`（または同等 CI）で schema ドリフトが検出される。
- AC04: generated schema が `version`, `git.commit`, `step.on_fail`, `step.uses`, `run/uses XOR` を表現する。
- AC05: unknown fields 非許容仕様が runtime + schema の双方で維持される。

## Decision Trace

- DEC01 -> ADR-0001 (`docs/adr/0001-go-runner-over-shell-generation.md`)
- DEC02 -> ADR-0002 (`docs/adr/0002-generate-config-schema-from-go-types.md`)

## Design -> Task Trace Matrix

- GOAL01 -> T1, T3
- GOAL02 -> T3, T4
- GOAL03 -> T2
- GOAL04 -> T1, T2, T4
- NONGOAL01 -> T2 guard
- NONGOAL02 -> T2 guard
- NONGOAL03 -> T4 guard
- DEC01 -> T1, T2
- DEC02 -> T1, T2, T3, T4
- REQ01 -> T1, T4
- REQ02 -> T1
- REQ03 -> T2
- REQ04 -> T1
- REQ05 -> T3
- REQ06 -> T3
- REQ07 -> T2
- REQ08 -> T4
- AC01 -> T1, T4
- AC02 -> T3
- AC03 -> T3
- AC04 -> T2
- AC05 -> T2

## Task -> Design Compose Matrix

- T1: GOAL01, GOAL04, DEC01, DEC02, REQ01, REQ02, REQ04, AC01
- T2: GOAL03, GOAL04, DEC01, DEC02, REQ03, REQ07, AC04, AC05, NONGOAL01, NONGOAL02
- T3: GOAL01, GOAL02, DEC02, REQ05, REQ06, AC02, AC03
- T4: GOAL02, GOAL04, DEC02, REQ01, REQ08, AC01, NONGOAL03

## Full Cross Self-Check Evidence

### Forward Fidelity (Design -> Tasks)

- Coverage ratio (`REQ+AC covered / total REQ+AC`): `13/13`
- Coverage ratio (`DEC covered / total DEC`): `2/2`
- `GOALxx` coverage (`covered / total`): `4/4`
- Invalid DEC-to-ADR mappings: none
- Missing design atoms: none
- Verdict: PASS

### Reverse Fidelity (Tasks -> Design)

- Orphan tasks (no valid anchors): none
- Tasks missing `REQxx/ACxx` in `Satisfied Requirements`: none
- Tasks referencing unknown design atoms: none
- Reconstructed intent matches design scope/acceptance: yes
- Alignment verdict: PASS
- Gaps and actions: none

### Non-Goal Guard

- Violations against `NONGOALxx`: none
- Notes: NONGOAL01/NONGOAL02 は T2 の DoD で、NONGOAL03 は T4 の DoD で明示ガード。
- Verdict: PASS

### DoD Semantics Guard

- Tasks with OR-like DoD wording: none
- DoD items missing independent verification: none
- Notes: T1-T4 すべて "All DoD items are mandatory AND conditions" を明示。
- Verdict: PASS

### Granularity Guard

- Tasks too broad for a single coherent change unit: none
- Tasks too fragmented (should be merged): none
- Notes: 生成基盤、制約反映、CI 統合、運用ガードの 4 単位に分離。
- Verdict: PASS

### Round-Trip Gate

- Alignment verdict: PASS
- Required fixes: none
- Checked At: 2026-02-24
