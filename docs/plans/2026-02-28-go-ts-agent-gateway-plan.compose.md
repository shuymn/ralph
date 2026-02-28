# Go + TS Agent Gateway Integration Plan Compose Pack

**Input Design**: `docs/plans/2026-02-28-go-ts-agent-gateway-design.md`
**Task Source**: `docs/plans/2026-02-28-go-ts-agent-gateway-plan.md`

## Compose Reconstruction

### Reconstructed Design Summary
- `run/review/judge` の main 実行は shell command 経路を廃止し、gateway 経路へ統一される。
- Go は control plane、TS は execution plane として分離され、provider SDK 差分は TS gateway 側へ局所化される。
- TS 側で stdio JSON-RPC server（`initialize/run_turn/cancel_turn/shutdown`）と provider adapters（`codex/claude`）を実装し、stream 正規化と duplicate suppression を行う。
- TS 初期化時に `task check` から `bun run check`（format/lint/typecheck）を実行する経路を組み込む。
- Go 側で protocol decode/validation、stream 契約検証、runner 統合を実装し、structured output 必須契約違反を runtime error (`exit 22`) へ fail-closed で接続する。
- gateway artifact は bun compile + archive + sha256 + go:embed + cache materialize で配布され、非対応 platform は明示エラーとなる。
- provider 解決は role 優先・run fallback で deterministic に動作し、dry-run で source trace を出力する。
- protocol 契約は Go/TS shared fixtures で互換検証され、standalone protocol jsonschema は追加しない。
- TEMP01/TEMP02 は introduce/migrate/retire タスクを保持しつつ、今回実行は deferred（waiver 付き）として管理される。

### Scope Diff
- Missing from tasks: none
- Extra in tasks: none
- Ambiguous mappings: none
- Open temporary mechanisms (`TEMPxx`):
  - TEMP01: status=`waived` (reason=`user requested no TEMP01 execution in current plan`, deadline=`2026-Q2`, owner=`@ralph-maintainers`)
  - TEMP02: status=`waived` (reason=`user requested no TEMP02 execution in current plan`, deadline=`2026-Q2`, owner=`@ralph-maintainers`)

### Alignment Verdict
- PASS
- Required fixes: none
