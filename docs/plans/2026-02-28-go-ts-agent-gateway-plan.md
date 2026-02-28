# Go + TS Agent Gateway Integration Implementation Plan

**Source**: `docs/plans/2026-02-28-go-ts-agent-gateway-design.md`
**Trace Pack**: `docs/plans/2026-02-28-go-ts-agent-gateway-plan.trace.md`
**Compose Pack**: `docs/plans/2026-02-28-go-ts-agent-gateway-plan.compose.md`
**Goal**: `run/review/judge` の main 実行を gateway 経路へ統一し、Go control plane と TS execution plane の責務境界を維持しながら v1 契約を fail-closed で満たす。
**Architecture**: Go は config/loop/completion/git/exit code と gateway lifecycle を担い、TS gateway は provider SDK 実行と stream 正規化を担う。通信は stdio JSON-RPC、契約整合は Go/TS 型定義 + shared fixtures で検証する。
**Tech Stack**: Go (`internal/config`, `internal/runner`, `internal/gateway`), TypeScript gateway (bun), JSON-RPC over stdio, go:embed, Task.

## Task Dependency Graph

`T1 -> T8 -> T7 -> T9 -> T10`
`T1 -> T2 -> T3 -> T4 -> T5 -> T6 -> T7`
`T3 -> T10`
`T2 -> T11 -> T12`
`T9 -> T13 -> T14`

## Task List

### Task 1: Gateway Config Model Cutover

**Satisfied Requirements**: REQ07, REQ08, AC01
**Design Anchors**: GOAL01, REQ07, REQ08, AC01, DEC08
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`
**Goal**: legacy command 設定を廃止し、gateway/role_provider/providers を SoT とする config モデルへ移行する。
**Dependencies**: none

**Files:**
- Modify: `internal/config/types.go` (gateway/role_provider/providers 定義)
- Modify: `internal/config/types_yaml.go` (YAML tag 更新)
- Modify: `internal/config/validate.go` (legacy key reject + enum/run必須)
- Modify: `internal/config/load_test.go` (load/validation 回帰)
- Modify: `internal/config/schema/generator.go` (schema 制約反映)
- Modify: `internal/config/schema/generator_test.go` (schema 回帰)
- Modify: `schemas/config.schema.json` (生成物同期)
- Modify: `internal/init/templates/config.tmpl` (scaffold 同期)

**RED**
- legacy command keys を受理してしまう失敗テストを追加する。
- `transport != stdio`、`role_provider.run` 未指定、未知 provider を許容してしまう失敗テストを追加する。
- env override が有効になってしまう失敗テストを追加する。
- Run: `go test ./internal/config ./internal/config/schema -run 'TestLoadRejectsLegacyAgentCommandKeys|TestValidateGatewayTransportAndProviderEnums|TestLoadRejectsMissingRunProvider|TestLoadIsConfigOnlyWithoutEnvOverrides|TestSchemaIncludesGatewayRoleProviderFields'`
- Expected: `FAIL with assertion/runtime mismatch against gateway config contract`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- gateway 中心の config 型/validation/schema を揃える。
- `.ralph/config.yml` 単一 SoT を維持し、env override を導入しない。

**REFACTOR**
- default 補完と validation 境界を整理し、将来の provider 拡張時の回帰点を最小化する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- legacy command keys を含む config は load 失敗する（REQ08, AC01）。
- `agent.gateway.transport=stdio`、`role_provider.run` 必須、provider enum 制約が load/validate/schema で一致する（REQ07, REQ08）。
- config 解決は `.ralph/config.yml` 単一 SoT で動作し、env override を導入しない（REQ07）。
- Run: `go test ./internal/config/... ./internal/config/schema/... && task schema && git diff --exit-code -- schemas/config.schema.json`
- Expected: `PASS`

### Task 2: Bundled Gateway Materialization and Platform Guard (TEMP01 Introduce)

**Satisfied Requirements**: REQ13, REQ14, AC06, AC08
**Design Anchors**: GOAL03, GOAL04, REQ13, REQ14, AC06, AC08, DEC07, TEMP01
**Temporary Mechanisms**: Creates `TEMP01`, Retires `none`, Waiver `{reason: "TEMP01 migrate/retire は今回の実装対象外", deadline: "2026-Q2", owner: "@ralph-maintainers"}`
**Goal**: embed artifact の展開・checksum 検証・platform guard を実装し、手動起動不要の前提を整備する。
**Dependencies**: T1

**Files:**
- Create: `internal/gateway/bundle_manifest.go` (embedded metadata)
- Create: `internal/gateway/materialize.go` (cache 展開 + checksum 検証)
- Create: `internal/gateway/platform.go` (supported platform 判定)
- Create: `internal/gateway/materialize_test.go` (materialize/checksum/cleanup)
- Create: `internal/gateway/platform_test.go` (unsupported platform)
- Modify: `main.go` (materialize 初期化)

**RED**
- checksum 不一致を検知できない失敗テストを追加する。
- `darwin/arm64` 以外で `errNoBundledGatewayForPlatform` を返せない失敗テストを追加する。
- cache 再利用に失敗する失敗テストを追加する。
- Run: `go test ./internal/gateway -run 'TestMaterializeGatewayVerifiesChecksum|TestMaterializeGatewayReusesCachedBinary|TestUnsupportedPlatformReturnsErrNoBundledGatewayForPlatform'`
- Expected: `FAIL with assertion/runtime mismatch for packaging guard`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- `go:embed` artifact を `UserCacheDir` に展開し、checksum 検証済み binary のみ実行対象にする。
- 非対応 platform を明示エラーで fail-closed する。

**REFACTOR**
- bundle 解決と materialize 実処理の責務を分離し、multi-platform 追加時の差分を局所化する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- checksum 検証済み artifact のみ実行対象となる（REQ13）。
- 非対応 platform では `errNoBundledGatewayForPlatform` で失敗する（REQ14, AC08）。
- `ralph run/review` 実行前に手動 gateway 起動を要求しない（AC06）。
- Run: `go test ./internal/gateway -run 'TestMaterializeGatewayVerifiesChecksum|TestMaterializeGatewayReusesCachedBinary|TestUnsupportedPlatformReturnsErrNoBundledGatewayForPlatform'`
- Expected: `PASS`

### Task 3: TypeScript Gateway Workspace Bootstrap (bun-template Baseline)

**Satisfied Requirements**: REQ15
**Design Anchors**: GOAL04, REQ15, DEC07
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`
**Goal**: TS gateway workspace を作成し、Bun セットアップは `shuymn/bun-template` を基準として統一する。
**Dependencies**: T2

**Files:**
- Create: `gateway/package.json` (bun scripts)
- Create: `gateway/bunfig.toml` (compile 設定)
- Create: `gateway/tsconfig.json` (TS compile 設定)
- Create: `gateway/src/main.ts` (entrypoint)
- Create: `gateway/test/` (bun test 土台)
- Create: `internal/gateway/build_pipeline_test.go` (gateway workspace/build precondition tests)
- Modify: `Taskfile.yml` (gateway build/test tasks)

**RED**
- gateway workspace 未作成でも build pipeline が通ってしまう失敗テストを追加する。
- Bun スクリプト規約不整合を検知できない失敗テストを追加する。
- `task check` 実行時に `bun run check` が呼ばれない失敗テストを追加する。
- Run: `go test ./internal/gateway -run 'TestGatewayBuildRequiresTSWorkspace|TestGatewayBuildScriptsFollowTemplateBaseline|TestTaskCheckInvokesBunCheck'`
- Expected: `FAIL with missing gateway workspace/script assertions`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- Bun workspace を初期化し、`package.json`/`bunfig.toml`/script 構成を [shuymn/bun-template](https://github.com/shuymn/bun-template) 基準で整える。
- `task check` が `bun run check` を呼ぶよう Taskfile を接続し、TS 側の format/lint/typecheck を初期化時から検証できるようにする。

**REFACTOR**
- build/test script 名称を統一し、CI とローカルで同一コマンドを再利用できるようにする。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- gateway workspace が存在し、Task 経由で Bun build/test を起動できる（REQ15）。
- Bun セットアップは [shuymn/bun-template](https://github.com/shuymn/bun-template) を基準にし、差分は gateway 要件に必要な理由を説明可能である（REQ15）。
- `task check` で `bun run check` が実行され、`bun-template` の `check` script（format/lint/typecheck）を通る（REQ15）。
- Run: `go test ./internal/gateway -run 'TestTaskCheckInvokesBunCheck' && task check`
- Expected: `PASS`

### Task 4: TypeScript JSON-RPC Server and Turn Contract

**Satisfied Requirements**: REQ03, REQ04, AC03, AC05
**Design Anchors**: GOAL02, REQ03, REQ04, AC03, AC05, DEC03, DEC04, DEC06
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`
**Goal**: TS gateway 側で `initialize/run_turn/cancel_turn/shutdown` を実装し、全 role の structured output 必須契約を enforce する。
**Dependencies**: T3

**Files:**
- Create: `gateway/src/protocol/types.ts` (request/response/notification)
- Create: `gateway/src/server/jsonrpc_stdio.ts` (stdio JSON-RPC server)
- Create: `gateway/src/server/handlers.ts` (method handlers)
- Create: `gateway/test/jsonrpc_server.test.ts` (method/contract tests)

**RED**
- required methods の欠落を検知する失敗テストを追加する。
- `run_turn.output_schema` 欠落、`final_result.structured_output` 欠落を許容してしまう失敗テストを追加する。
- Run: `bun test gateway/test/jsonrpc_server.test.ts`
- Expected: `FAIL with method/contract assertion mismatch`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- stdio JSON-RPC server と method handlers を実装する。
- `run_turn.output_schema`/`final_result.structured_output` 必須検証を role 共通で実装する。

**REFACTOR**
- protocol decode/validate 層を分離し、handler が domain エラーへ集中できる構造に整理する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- TS gateway が `initialize/run_turn/cancel_turn/shutdown` を受理する（REQ03, AC05）。
- `run_turn.output_schema` と `final_result.structured_output` は全 role で必須化される（REQ04, AC03）。
- Run: `bun test gateway/test/jsonrpc_server.test.ts`
- Expected: `PASS`

### Task 5: TypeScript Provider Adapters and Unified Stream Normalization

**Satisfied Requirements**: REQ02, REQ05, AC05
**Design Anchors**: GOAL02, REQ02, REQ05, AC05, DEC01, DEC06
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`
**Goal**: `codex/claude` provider adapter と stream 正規化を実装し、Go 側が単一イベント契約のみ扱える状態にする。
**Dependencies**: T4

**Files:**
- Create: `gateway/src/providers/codex_adapter.ts` (codex adapter)
- Create: `gateway/src/providers/claude_adapter.ts` (claude adapter)
- Create: `gateway/src/stream/normalizer.ts` (event normalization + duplicate suppression)
- Create: `gateway/test/stream_normalizer.test.ts` (seq/duplicate/event tests)

**RED**
- provider 切替ができない失敗テストを追加する。
- `turn_id/seq` 契約違反や重複 emit を検知できない失敗テストを追加する。
- TS 側へ completion 判定ロジックが混入しても検知できない失敗テストを追加する。
- Run: `bun test gateway/test/stream_normalizer.test.ts`
- Expected: `FAIL with provider/stream contract assertion mismatch`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- codex/claude adapter を実装し、共通 stream event 契約へ正規化する。
- `turn_id/seq` 契約と duplicate suppression を gateway で実装する。

**REFACTOR**
- adapter interface を統一し、provider 差分が transport 層へ漏れないようにする。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- Go は provider SDK を直呼びせず、TS execution plane が provider 差分吸収を担う（REQ02）。
- stream 契約（`turn_id/seq`、duplicate suppression）を TS gateway 側で満たす（REQ05, AC05）。
- Run: `bun test gateway/test/stream_normalizer.test.ts && ! rg -n 'tail_match|review_convergence|completeReview|exit code' gateway/src`
- Expected: `PASS`

### Task 6: Go Gateway Client and Protocol Validation

**Satisfied Requirements**: REQ03, REQ04, REQ05, AC03, AC05
**Design Anchors**: GOAL02, REQ03, REQ04, REQ05, AC03, AC05, DEC03, DEC04, DEC06
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`
**Goal**: Go 側 gateway client を実装し、protocol violation を fail-closed で検出する。
**Dependencies**: T2, T4, T5

**Files:**
- Create: `internal/gateway/rpc_client.go` (request/response/cancel/shutdown)
- Create: `internal/gateway/protocol_types.go` (Go protocol types)
- Create: `internal/gateway/stream_validator.go` (`turn_id/seq` validation)
- Create: `internal/gateway/rpc_client_test.go` (contract tests)
- Create: `internal/gateway/stream_validator_test.go` (stream tests)

**RED**
- invalid message を protocol violation として reject できない失敗テストを追加する。
- `stream_event` 順序違反/必須欠落を許容してしまう失敗テストを追加する。
- Run: `go test ./internal/gateway -run 'TestRunTurnRequiresOutputSchema|TestRunTurnRequiresStructuredOutputForAllRoles|TestRejectsInvalidProtocolMessages|TestStreamEventRequiresTurnIDAndMonotonicSeq'`
- Expected: `FAIL with protocol assertion/runtime mismatch`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- Go 側 protocol decode/validate と RPC client を実装する。
- violation を runtime error 経路へ接続する。

**REFACTOR**
- typed error を整理し、runner 側 exit code 分岐を明確化する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- Go 側で `run_turn.output_schema`/`final_result.structured_output` 必須契約を検証できる（REQ04, AC03）。
- invalid protocol message は fail-closed で拒否される（REQ03, REQ05, AC05）。
- Run: `go test ./internal/gateway -run 'TestRunTurnRequiresOutputSchema|TestRunTurnRequiresStructuredOutputForAllRoles|TestRejectsInvalidProtocolMessages|TestStreamEventRequiresTurnIDAndMonotonicSeq'`
- Expected: `PASS`

### Task 7: Runner Main-Step Gateway Path and Structured Output Persistence

**Satisfied Requirements**: REQ01, REQ09, REQ10, REQ11, REQ17, AC01, AC04, AC06
**Design Anchors**: GOAL01, GOAL05, REQ01, REQ09, REQ10, REQ11, REQ17, AC01, AC04, AC06, DEC01, DEC02, DEC04
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`
**Goal**: main step を gateway 経路へ置換し、artifact 保存・completion 判定・exit code 契約を維持する。
**Dependencies**: T6, T8

**Files:**
- Modify: `internal/runner/main_step.go` (gateway only path)
- Modify: `internal/runner/loop.go` (mode completion 維持)
- Modify: `internal/runner/review_artifacts.go` (REVIEW/JUDGE persist)
- Modify: `internal/runner/judge_input.go` (judge input contract)
- Modify: `internal/runner/loop_test.go` (main-step behavior)

**RED**
- shell command 経路へフォールバックする失敗テストを追加する。
- `structured_output` 欠落時に fallback parse してしまう失敗テストを追加する。
- `run/review` の `output_text` 保存、`judge` の `JUDGE_n.json` 保存が崩れる失敗テストを追加する。
- Run: `go test ./internal/runner -run 'TestRunMainStepUsesGatewayOnlyPath|TestRunReviewPersistStructuredOutputTextForCompletion|TestJudgePersistsStructuredOutputAsJudgeArtifact|TestMissingStructuredOutputReturnsRuntimeErrorExit22|TestRunReviewUXDoesNotRequireManualGatewayStart'`
- Expected: `FAIL with runner contract assertion/runtime mismatch`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- shell command 経路を廃止し gateway 経路を唯一化する。
- `run/review/judge` の structured output 永続化契約を維持する。
- fallback parse を廃止し、違反時 runtime error (`exit 22`) を返す。

**REFACTOR**
- role 別永続化ロジックを整理し、main step 分岐の重複を削減する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `run/review` の main 実行は gateway 経路のみとなる（REQ01, AC01）。
- `run/review` は `structured_output.output_text`、`judge` は `structured_output` を既存規約で保存する（REQ09, REQ10, AC04）。
- structured output 欠落/不一致は fallback parse せず runtime error (`exit 22`) となる（REQ11）。
- CLI UX は `ralph run/review` を維持し、手動 gateway 起動を要求しない（REQ17, AC06）。
- Run: `go test ./internal/runner -run 'TestRunMainStepUsesGatewayOnlyPath|TestRunReviewPersistStructuredOutputTextForCompletion|TestJudgePersistsStructuredOutputAsJudgeArtifact|TestMissingStructuredOutputReturnsRuntimeErrorExit22|TestRunReviewUXDoesNotRequireManualGatewayStart'`
- Expected: `PASS`

### Task 8: Role-aware Provider Resolution and Dry-Run Trace

**Satisfied Requirements**: REQ06, AC02
**Design Anchors**: GOAL01, REQ06, AC02, DEC05
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`
**Goal**: role 優先 + run fallback の provider 解決と dry-run source trace を実装する。
**Dependencies**: T1

**Files:**
- Modify: `internal/runner/mode_plan.go` (resolve order)
- Modify: `internal/runner/dryrun.go` (resolved/source output)
- Modify: `internal/runner/dryrun_test.go` (resolution tests)

**RED**
- role 指定優先が守られない失敗テストを追加する。
- `review/judge` 未指定時 fallback が働かない失敗テストを追加する。
- Run: `go test ./internal/runner -run 'TestResolveProviderPrefersRoleSpecific|TestResolveProviderFallsBackToRun|TestDryRunReportsResolvedProviderAndSource'`
- Expected: `FAIL with provider resolution assertion mismatch`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- deterministic provider 解決順序を実装する。
- dry-run で resolved/source を表示する。

**REFACTOR**
- provider 解決ヘルパーを統一し、mode 分岐の重複を削減する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- `review/judge` 未指定時は `run` provider へ fallback する（REQ06, AC02）。
- role 指定がある場合は role 指定が優先される（REQ06）。
- dry-run が `provider.<role>.resolved`/`source` を表示する（REQ06）。
- Run: `go test ./internal/runner -run 'TestResolveProviderPrefersRoleSpecific|TestResolveProviderFallsBackToRun|TestDryRunReportsResolvedProviderAndSource'`
- Expected: `PASS`

### Task 9: Retry/Timeout Taxonomy and Best-Effort Cancel (TEMP02 Introduce)

**Satisfied Requirements**: REQ12
**Design Anchors**: GOAL03, REQ12, DEC09, TEMP02
**Temporary Mechanisms**: Creates `TEMP02`, Retires `none`, Waiver `{reason: "TEMP02 migrate/retire は今回の実装対象外", deadline: "2026-Q2", owner: "@ralph-maintainers"}`
**Goal**: timeout 分離と retry classification に基づく単回 retry、signal 時 cancel を導入する。
**Dependencies**: T6, T7

**Files:**
- Create: `internal/gateway/retry_policy.go` (retry classification)
- Modify: `internal/gateway/rpc_client.go` (timeout + retry)
- Modify: `internal/runner/main_step.go` (retry reason log)
- Modify: `internal/gateway/rpc_client_test.go` (retry/timeout tests)
- Modify: `internal/runner/loop_test.go` (signal cancel regression test)

**RED**
- retryable/non-retryable の分類どおりに動作しない失敗テストを追加する。
- idle timeout と signal cancel を検知できない失敗テストを追加する。
- Run: `go test ./internal/gateway ./internal/runner -run 'TestGatewayCrashRetriesOnceWithBackoff|TestValidationAndSchemaViolationsAreNotRetried|TestIdleTimeoutRetriesOnce|TestSignalPathSendsCancelTurnBestEffort'`
- Expected: `FAIL with retry/timeout assertion mismatch`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- `startup/rpc/idle` timeout を分離し、同一 turn 最大1回 retry + backoff を実装する。
- retry reason を debug ログ化し、signal 終了時に `cancel_turn` を best-effort 送信する。

**REFACTOR**
- retry 判定テーブルをデータ駆動化し、分類拡張時の差分を小さくする。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- timeout が `startup/rpc/idle` で分離される（REQ12）。
- retryable は同一 turn 最大1回再試行し、non-retryable は再試行しない（REQ12）。
- signal 終了時に `cancel_turn` が best-effort で送信される（REQ12）。
- Run: `go test ./internal/gateway ./internal/runner -run 'TestGatewayCrashRetriesOnceWithBackoff|TestValidationAndSchemaViolationsAreNotRetried|TestIdleTimeoutRetriesOnce|TestSignalPathSendsCancelTurnBestEffort'`
- Expected: `PASS`

### Task 10: Shared Protocol Fixture Compatibility (Go + TS)

**Satisfied Requirements**: REQ16, AC07
**Design Anchors**: GOAL02, REQ16, AC07, DEC03
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `none`
**Goal**: shared fixtures を Go/TS 双方で実行し、protocol 契約の乖離を検知可能にする。
**Dependencies**: T3, T4, T5, T6

**Files:**
- Create: `internal/gateway/testdata/protocol/valid/*.json` (valid fixtures)
- Create: `internal/gateway/testdata/protocol/invalid/*.json` (invalid fixtures)
- Create: `internal/gateway/protocol_fixture_test.go` (Go fixture runner)
- Create: `gateway/test/protocol_fixtures.test.ts` (TS fixture runner)
- Modify: `Taskfile.yml` (`test:protocol`)

**RED**
- Go/TS 片側のみで fixture 実行しても検知できない失敗テストを追加する。
- invalid fixture を protocol violation として reject できない失敗テストを追加する。
- Run: `task test:protocol`
- Expected: `FAIL with fixture parity or invalid-message rejection mismatch`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- shared fixture セットを整備し、Go/TS 双方で同一ケースを評価する。
- invalid message を protocol violation として fail-closed で扱う。

**REFACTOR**
- fixture naming/versioning 規約を統一し、`protocol_version` 更新時の差分管理を簡素化する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- protocol SoT は Go/TS 型定義 + shared fixtures で維持される（REQ16, AC07）。
- invalid fixture は Go/TS 双方で protocol violation として reject される（REQ16, AC07）。
- standalone protocol jsonschema artifact を追加しない（REQ16）。
- Run: `task test:protocol`
- Expected: `PASS`

### Task 11: TEMP01 Migrate/Cutover to Multi-Platform Bundles (Deferred)

**Satisfied Requirements**: REQ18
**Design Anchors**: REQ18, DEC09, TEMP01
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `{reason: "user requested no TEMP01 execution in this plan", deadline: "2026-Q2", owner: "@ralph-maintainers"}`
**Goal**: multi-platform bundle 解決へ切り替える cutover を将来実行可能な単位で保持する。
**Dependencies**: T2, T3

**Files:**
- Create: `internal/gateway/platform_bundle_resolver.go` (multi-platform resolver)
- Modify: `internal/gateway/platform_test.go` (linux/windows cases)
- Modify: `.github/workflows/ci.yml` (matrix expansion)

**RED**
- `linux/amd64`, `linux/arm64`, `windows/amd64` bundle 解決失敗を再現する失敗テストを追加する。
- Run: `go test ./internal/gateway -run 'TestResolveGatewayBundleForLinuxAndWindowsTargets'`
- Expected: `FAIL with missing multi-platform cutover assertions`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- platform ごとの bundle 解決と CI matrix を導入する。

**REFACTOR**
- platform table 駆動へ統一し、target 追加差分を局所化する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- multi-platform bundle 解決が実装され、対象 OS/arch CI が green になる（REQ18）。
- Run: `go test ./internal/gateway -run 'TestResolveGatewayBundleForLinuxAndWindowsTargets'`
- Expected: `PASS`

### Task 12: TEMP01 Retire Darwin-only Branches (Deferred)

**Satisfied Requirements**: REQ18
**Design Anchors**: REQ18, DEC09, TEMP01
**Temporary Mechanisms**: Creates `none`, Retires `TEMP01`, Waiver `{reason: "depends on T11 and sunset trigger", deadline: "2026-Q2", owner: "@ralph-maintainers"}`
**Goal**: darwin-only 分岐と single-bundle 前提を削除し、TEMP01 を退役させる。
**Dependencies**: T11

**Files:**
- Modify: `internal/gateway/platform.go` (darwin-only branch removal)
- Modify: `internal/gateway/bundle_manifest.go` (single-bundle assumption removal)
- Modify: `internal/gateway/platform_test.go` (negative fallback tests)

**RED**
- 旧 darwin-only fallback が残存する失敗テストを追加する。
- Run: `go test ./internal/gateway -run 'TestDarwinOnlyPlatformBranchIsRemovedAfterTemp01Retirement'`
- Expected: `FAIL with residual darwin-only fallback assertions`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- darwin-only 分岐を削除し、multi-platform resolver を唯一経路にする。

**REFACTOR**
- dead code を除去し、support target 定義を集約する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- TEMP01 removal scope（darwin-only 分岐 + single-bundle 前提）削除が完了する（REQ18）。
- 否定検証として旧 fallback 経路が呼ばれないことを確認する（REQ18）。
- Run: `go test ./internal/gateway -run 'TestDarwinOnlyPlatformBranchIsRemovedAfterTemp01Retirement'`
- Expected: `PASS`

### Task 13: TEMP02 Migrate/Cutover to Classified Retry Policy (Deferred)

**Satisfied Requirements**: REQ19
**Design Anchors**: REQ19, DEC09, TEMP02
**Temporary Mechanisms**: Creates `none`, Retires `none`, Waiver `{reason: "user requested no TEMP02 execution in this plan", deadline: "2026-Q2", owner: "@ralph-maintainers"}`
**Goal**: protocol classification 駆動 retry policy へ切り替える移行単位を保持する。
**Dependencies**: T9, T10

**Files:**
- Modify: `internal/gateway/protocol_types.go` (classification fields)
- Modify: `internal/gateway/retry_policy.go` (policy table)
- Modify: `internal/gateway/rpc_client_test.go` (classification tests)
- Modify: `internal/gateway/testdata/protocol/*` (classification fixtures)

**RED**
- classification 導入後も固定 retry 分岐へ依存する失敗テストを追加する。
- Run: `go test ./internal/gateway -run 'TestRetryPolicyUsesProtocolClassificationTable|TestProtocolFixturesCoverRetryClassification'`
- Expected: `FAIL with residual fixed-retry path assertions`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- protocol classification を取り込み、retry policy table で挙動を決定する。

**REFACTOR**
- classification の定義と policy mapping を1箇所に集約する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- retry 判定が protocol classification 駆動になる（REQ19）。
- classification 別テストが追加され、固定 retry 分岐への暗黙依存を除去できる（REQ19）。
- Run: `go test ./internal/gateway -run 'TestRetryPolicyUsesProtocolClassificationTable|TestProtocolFixturesCoverRetryClassification'`
- Expected: `PASS`

### Task 14: TEMP02 Retire Fixed One-Retry Branch (Deferred)

**Satisfied Requirements**: REQ19
**Design Anchors**: REQ19, DEC09, TEMP02
**Temporary Mechanisms**: Creates `none`, Retires `TEMP02`, Waiver `{reason: "depends on T13 and production validation", deadline: "2026-Q2", owner: "@ralph-maintainers"}`
**Goal**: 固定1回 retry 分岐を削除し、classification 駆動 policy のみを残す。
**Dependencies**: T13

**Files:**
- Modify: `internal/gateway/rpc_client.go` (fixed retry branch removal)
- Modify: `internal/gateway/retry_policy.go` (classified policy only)
- Modify: `internal/gateway/rpc_client_test.go` (negative branch tests)

**RED**
- fixed one-retry 分岐が残っても検知できない失敗テストを追加する。
- Run: `go test ./internal/gateway -run 'TestFixedOneRetryBranchIsRemovedAfterTemp02Retirement'`
- Expected: `FAIL with residual fixed-retry branch assertions`
- Note: compilation/import/module errors are not valid RED outcomes.

**GREEN**
- fixed one-retry 分岐を削除し、classification table を唯一経路にする。

**REFACTOR**
- retry log 出力を policy table と一致する形式へ整理する。

**DoD**
- All DoD items are mandatory AND conditions (never OR).
- TEMP02 removal scope（固定1回 retry 分岐削除）が完了する（REQ19）。
- 否定検証として固定分岐が実行されないことを確認する（REQ19）。
- Run: `go test ./internal/gateway -run 'TestFixedOneRetryBranchIsRemovedAfterTemp02Retirement'`
- Expected: `PASS`

## Checkpoint Summary

- Alignment Verdict: PASS
- Forward Fidelity: PASS
- Reverse Fidelity: PASS
- Non-Goal Guard: PASS
- Granularity Guard: PASS
- Temporal Completeness Guard: PASS
- TEMP Summary: introduced=2, retired=0, open=2, waived=2
- Trace Pack: `docs/plans/2026-02-28-go-ts-agent-gateway-plan.trace.md`
- Compose Pack: `docs/plans/2026-02-28-go-ts-agent-gateway-plan.compose.md`
- Updated At: `2026-02-28`
