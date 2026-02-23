# Config Schema Sync Plan Compose Pack

## Compose Reconstruction

### Reconstructed Design Summary

- `internal/config` の型定義を起点に `schemas/config.schema.json` を生成し、手動更新を廃止する。
- schema は構造契約を担い、runtime の意味制約は既存 `Validate` を維持する。
- generated schema には runtime と整合する最小制約（const/enum/XOR）を post-process で付与する。
- `task schema` を再生成の標準入口とし、`task check` でドリフトを検知する。
- unknown fields 非許容は runtime (`KnownFields(true)`) と schema (`additionalProperties: false`) の双方で維持する。
- `.ralph/` 配下に schema 実ファイルを展開しない方針を維持し、運用ガイドにも明記する。

### Scope Diff

- Missing from tasks: none
- Extra in tasks: none
- Ambiguous mappings: none

### Alignment Verdict

- PASS
- Required fixes: none
- Checked At: 2026-02-24
