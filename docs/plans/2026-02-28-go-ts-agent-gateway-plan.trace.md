# Go + TS Agent Gateway Integration Plan Trace Pack

**Input Design**: `docs/plans/2026-02-28-go-ts-agent-gateway-design.md`

## Design Atom Index

### Goals
- GOAL01: `run/review` の main 実行を gateway 経路で統一する。
- GOAL02: SDK/provider 差分を TS gateway 側へ局所化し、Go は共通契約のみ扱う。
- GOAL03: 手動で gateway を先に起動しない UX を実現する。
- GOAL04: Node 依存なし配布のため bun native compile artifact を embed 利用する。
- GOAL05: completion / review convergence / exit code 体系を維持する。

### Non-Goals
- NONGOAL01: v1 で Unix socket transport を導入しない。
- NONGOAL02: v1 で `darwin/arm64` 以外を配布対象にしない。
- NONGOAL03: 旧 command 設定との互換レイヤーを維持しない。
- NONGOAL04: v1 で gateway 常駐デーモン管理コマンドを追加しない。

### Requirements
- REQ01: main step を gateway 経路へ置換し、shell command 実行経路を廃止する。
- REQ02: Go は provider SDK を直呼びせず、TS は completion 判定ロジックを持たない責務境界を守る。
- REQ03: v1 transport は stdio JSON-RPC（`initialize/run_turn/cancel_turn/shutdown`）のみ。
- REQ04: `run_turn.output_schema` と `final_result.structured_output` を全 role で必須化する。
- REQ05: `stream_event` は `turn_id/seq` 契約と duplicate suppression を満たす。
- REQ06: provider 解決は role 優先、未指定時 run fallback とする。
- REQ07: config SoT は `.ralph/config.yml` のみで、env override を導入しない。
- REQ08: legacy command keys を廃止し、gateway/provider validation を fail-closed で行う。
- REQ09: prompt 入力規約（judge machine context prefix を含む）を維持する。
- REQ10: `run/review` は `structured_output.output_text`、`judge` は `structured_output` を `JUDGE_n.json` として保存する。
- REQ11: structured output 欠落/不一致は fallback parse せず runtime error (`exit 22`) とする。
- REQ12: timeout 分離、retry classification、単回 retry、cancel best-effort を実装する。
- REQ13: bun compile artifact を archive+sha256 で embed し、展開時 checksum を検証する。
- REQ14: 非対応 platform は `errNoBundledGatewayForPlatform` で明示失敗する。
- REQ15: gateway compile/archive/hash/embed を build/release pipeline へ組み込む。
- REQ16: protocol SoT は Go/TS 型定義 + shared fixtures とし、standalone protocol jsonschema artifact は追加しない。
- REQ17: UX は `ralph run/review` を維持し、`stream_event` は `stderr`、最終結果系は `stdout` 優先とする。
- REQ18: TEMP01 の migration/sunset 可能性を保持する。
- REQ19: TEMP02 の migration/sunset 可能性を保持する。

### Decisions
- DEC01: Go runner + TS gateway ハイブリッド構成を採用する。
- DEC02: completion/review convergence/exit code 判定は Go control plane で維持する。
- DEC03: protocol SoT は Go/TS 型定義 + shared fixtures で管理する。
- DEC04: structured output 必須 + fallback parse 不採用の fail-closed 方針を採用する。
- DEC05: role-aware provider 解決を deterministic に固定する。
- DEC06: v1 transport は stdio JSON-RPC のみ、`stream_event` は `stderr` 固定とする。
- DEC07: bun native compile gateway を embed 展開する。
- DEC08: `agent.run_command/review_command/judge_command` を廃止する。
- DEC09: TEMP01/TEMP02 を v1 で導入し、Sunset Clause に従い退役させる。

### Acceptance Criteria
- AC01: `run/review` の main 実行は gateway 経路のみで、legacy command keys は廃止される。
- AC02: provider 解決は role 優先、`review/judge` 未指定時は run fallback を使う。
- AC03: `run_turn.output_schema` と `final_result.structured_output` は全 role で必須。
- AC04: `run/review` は `output_text` を completion 入力へ使い、`judge` は `JUDGE_n.json` を永続化する。
- AC05: transport は `stdio + JSON-RPC` のみ、`stream_event` は `stderr` に出力される。
- AC06: ユーザーは手動 gateway 起動なしで `ralph run/review` を実行できる。
- AC07: protocol 契約は Go/TS 双方の shared fixtures で検証される。
- AC08: `darwin/arm64` 以外は `errNoBundledGatewayForPlatform` で明示失敗する。

### Temporary Mechanisms
- TEMP01:
  - mechanism: `darwin/arm64` 限定 bundled gateway 配布
  - retirement_trigger: `linux/amd64`, `linux/arm64`, `windows/amd64` 配布と CI 成功
  - retirement_verification: 対象 OS/arch で `ralph run --dry-run` と e2e が green
  - removal_scope: darwin-only 分岐と single-bundle 前提ロジック削除
  - closure_source: checklist
  - record_source: adr (`0006` Sunset Clause)
- TEMP02:
  - mechanism: gateway 異常終了時の固定「1回のみ」再試行
  - retirement_trigger: retryable/non-retryable 分類の protocol 導入
  - retirement_verification: retry policy table と分類別テスト追加
  - removal_scope: fixed one-retry 分岐削除と分類駆動 retry policy 置換
  - closure_source: checklist
  - record_source: adr (`0006` Sunset Clause)

## Decision Trace
- DEC01 -> ADR-0006 (`docs/adr/0006-embed-bun-agent-gateway-for-main-execution.md`)
- DEC02 -> ADR-0006 (`docs/adr/0006-embed-bun-agent-gateway-for-main-execution.md`)
- DEC03 -> ADR-0006 (`docs/adr/0006-embed-bun-agent-gateway-for-main-execution.md`)
- DEC04 -> ADR-0006 (`docs/adr/0006-embed-bun-agent-gateway-for-main-execution.md`)
- DEC05 -> ADR-0006 (`docs/adr/0006-embed-bun-agent-gateway-for-main-execution.md`)
- DEC06 -> ADR-0006 (`docs/adr/0006-embed-bun-agent-gateway-for-main-execution.md`)
- DEC07 -> ADR-0006 (`docs/adr/0006-embed-bun-agent-gateway-for-main-execution.md`)
- DEC08 -> ADR-0006 (`docs/adr/0006-embed-bun-agent-gateway-for-main-execution.md`)
- DEC09 -> ADR-0006 (`docs/adr/0006-embed-bun-agent-gateway-for-main-execution.md`)

## Design -> Task Trace Matrix
- GOAL01 -> T1, T7, T8
- GOAL02 -> T4, T5, T6, T10
- GOAL03 -> T2, T9
- GOAL04 -> T2, T3
- GOAL05 -> T7
- NONGOAL01 -> no task mapping (guarded by REQ03/T4/T6)
- NONGOAL02 -> no task mapping (guarded by REQ14/T2)
- NONGOAL03 -> no task mapping (guarded by REQ08/T1)
- NONGOAL04 -> no task mapping (guarded by task scope)
- REQ01 -> T7
- REQ02 -> T5
- REQ03 -> T4, T6
- REQ04 -> T4, T6
- REQ05 -> T5, T6
- REQ06 -> T8
- REQ07 -> T1
- REQ08 -> T1
- REQ09 -> T7
- REQ10 -> T7
- REQ11 -> T7
- REQ12 -> T9
- REQ13 -> T2
- REQ14 -> T2
- REQ15 -> T3
- REQ16 -> T10
- REQ17 -> T7
- REQ18 -> T11, T12
- REQ19 -> T13, T14
- AC01 -> T1, T7
- AC02 -> T8
- AC03 -> T4, T6
- AC04 -> T7
- AC05 -> T4, T5, T6
- AC06 -> T2, T7
- AC07 -> T10
- AC08 -> T2
- DEC01 -> T5, T7
- DEC02 -> T7
- DEC03 -> T4, T6, T10
- DEC04 -> T4, T6, T7
- DEC05 -> T8
- DEC06 -> T4, T5, T6
- DEC07 -> T2, T3
- DEC08 -> T1
- DEC09 -> T9, T11, T12, T13, T14
- TEMP01 -> T2 (introduce), T11 (migrate), T12 (retire)
- TEMP02 -> T9 (introduce), T13 (migrate), T14 (retire)

## Task -> Design Compose Matrix
- T1: REQ07, REQ08, AC01, GOAL01, DEC08
- T2: REQ13, REQ14, AC06, AC08, GOAL03, GOAL04, DEC07, TEMP01
- T3: REQ15, GOAL04, DEC07
- T4: REQ03, REQ04, AC03, AC05, GOAL02, DEC03, DEC04, DEC06
- T5: REQ02, REQ05, AC05, GOAL02, DEC01, DEC06
- T6: REQ03, REQ04, REQ05, AC03, AC05, GOAL02, DEC03, DEC04, DEC06
- T7: REQ01, REQ09, REQ10, REQ11, REQ17, AC01, AC04, AC06, GOAL01, GOAL05, DEC01, DEC02, DEC04
- T8: REQ06, AC02, GOAL01, DEC05
- T9: REQ12, GOAL03, DEC09, TEMP02
- T10: REQ16, AC07, GOAL02, DEC03
- T11: REQ18, DEC09, TEMP01
- T12: REQ18, DEC09, TEMP01
- T13: REQ19, DEC09, TEMP02
- T14: REQ19, DEC09, TEMP02

## Temporary Mechanism Trace
- TEMP01: introduced_by=[T2], migrated_by=[T11], retired_by=[T12], retirement_trigger=[`linux/amd64`,`linux/arm64`,`windows/amd64` bundle + CI success], retirement_verification=[target platform `ralph run --dry-run` + e2e green], removal_scope=[darwin-only branch and single-bundle assumptions removed], closure_source=checklist, record_source=adr, status=waived(reason="user requested no TEMP01 execution in current plan", deadline="2026-Q2", owner="@ralph-maintainers")
- TEMP02: introduced_by=[T9], migrated_by=[T13], retired_by=[T14], retirement_trigger=[protocol-level retry classification introduced], retirement_verification=[retry policy table + classified tests green], removal_scope=[fixed one-retry branch removed], closure_source=checklist, record_source=adr, status=waived(reason="user requested no TEMP02 execution in current plan", deadline="2026-Q2", owner="@ralph-maintainers")

## Cross Self-Check

### Forward Fidelity (Design -> Tasks)
- Coverage ratio (`REQ+AC covered / total REQ+AC`): `27/27`
- Coverage ratio (`REQ+AC in task DoD / total REQ+AC`): `27/27`
- Coverage ratio (`GOAL covered / total GOAL`): `5/5`
- Coverage ratio (`DEC covered / total DEC`): `9/9`
- Invalid DEC-to-ADR mappings: none
- Missing design atoms: none
- Verdict: PASS

### Reverse Fidelity (Tasks -> Design)
- Orphan tasks (no valid anchors): none
- Tasks missing `REQxx/ACxx` in `Satisfied Requirements`: none
- Tasks referencing unknown design atoms: none
- Reconstructed scope parity (via compose pack): maintained
- Alignment verdict: PASS
- Gaps and actions: none

### Non-Goal Guard
- Violations against `NONGOALxx`: none
- Additional behavior outside mapped design atoms: none
- Verdict: PASS

### DoD Semantics Guard
- Tasks with OR-like DoD wording: none
- DoD items missing independent verification: none
- RED invalidity (compile/import/module failures used as RED): none
- Verdict: PASS

### Granularity Guard
- Tasks too broad for one coherent change unit: none
- Tasks too fragmented (should be merged): none
- Split-signal triage:
  - TS workspace/bootstrap（T3）、TS runtime contract（T4）、TS provider normalization（T5）を分割し、検証フローを独立させた。
  - Go client/runner integration（T6/T7）と compatibility fixtures（T10）を分離し、rollback 境界を明確化した。
  - TEMP lifecycle は introduce/migrate/retire（T2/T11/T12, T9/T13/T14）に分割した。
- Waiver-needed unsplit tasks: none
- Verdict: PASS

### Temporal Completeness Guard
- TEMP entries missing introducing tasks: none
- TEMP entries missing migrating/cutover tasks: none
- TEMP entries missing retiring tasks: none
- Retire tasks missing negative fallback-removal verification: none
- TEMP entries missing in-doc closure summary (checklist/ledger row): none
- TEMP entries missing closure tuple fields (trigger/verification/removal_scope): none
- Open TEMP entries without waiver metadata (`reason`, `deadline`, `owner?`): none
- Verdict: PASS

### Round-Trip Gate
- Alignment verdict: PASS
- Required fixes: none
- Checked At: `2026-02-28`
