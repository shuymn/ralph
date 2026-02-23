# Ralph Runner - Design

## Overview

本ドキュメントは、既存の `ralph` 運用（`prd.json` + `prompt.md` + runner）を前提に、プロジェクト差分を YAML に閉じ込め、Go の runner が `.ralph/config.yml` を直接読み取りタスクループを実行する設計を定義する。

CLI は `ralph init`（雛形初期化）と `ralph run`（ループ実行）の 2 コマンドで構成する。初期導入を容易にするため、`ralph init` で `.ralph/config.yml`、`.ralph/prompt.md`、`.ralph/prd.json`、`.ralph/progress.md` の雛形を生成できるものとする。

`.ralph/prd.json`、`.ralph/progress.md`、`.ralph/.commit-msg` は実行時データとして扱う。`.ralph/` は private かつ disposable な作業領域であり、永続的な成果物（コード、テスト、docs/ADR）は通常のプロジェクト領域で管理する。

## Goals

- **G1.** `.ralph/config.yml` を入力に `ralph run` でタスクループを直接実行できること。
- **G2.** プロジェクト差分を YAML に集約し、runner のロジックを再利用できること。
- **G3.** 1 ループ = 1 タスク = 1 エージェント起動を維持し、N+1 起動を誘発しないこと。
- **G4.** `prd.json` は `deps` / `passes` を前提とした最小仕様のまま維持すること。
- **G5.** `prompt.md` を 1 枚固定で運用できること（初期化後は都度生成しない）。
- **G6.** 初期導入を容易にするため、`config.yml` / `prompt.md` / `prd.json` / `progress.md` の雛形を初期化時のみ生成できること。
- **G7.** `config.yml` は GitHub 公開 JSON Schema URL を参照し、YAML エディタ補完/検証をローカル schema 展開なしで利用できること。

## Non-Goals

- **NG1.** plan.md の機械抽出を前提としたタスク固有プロンプト生成を必須化しない。
- **NG2.** 蒸留（ADR 化、知見統合、重複排除）を runner の標準フローに組み込まない。
- **NG3.** `prd.json` に強いスキーマや DSL を導入しない。
- **NG4.** `profile` 切り替えによる複数モード生成（minimal / standard など）を v1 で導入しない。
- **NG5.** v1 で shell script 生成（`ralph generate`）を提供しない。runner ロジックは Go バイナリに内包する。
- **NG6.** `validate` など補助サブコマンドを v1 の必須要件にしない。
- **NG7.** `ralph init` 時に `.ralph/` 配下へ schema 実ファイル（例: `config.schema.json`）を展開しない。

## Background

現状の `ralph` は、`prd.json` の stories（`deps` / `passes`）管理、`prompt.md` での明確な DoD 指示、runner における quality gate と commit 自動化の責務分離がうまく機能している。

一方で、プロジェクトごとの差分（lint/fmt/test 構成、staging 対象、検証コマンド）を毎回手作業で埋め込むのは保守コストが高い。ここを YAML 化して再利用性を上げつつ、運用のシンプルさを維持する必要がある。

当初は YAML + Go `text/template` で shell script（`ralph.sh`）を生成する設計を検討した。しかしレビューの過程で以下の問題が明らかになった。

- shell script に汎用 runtime 関数（`if` 式評価、auto commit、completion check 等）を埋め込むと、プロジェクト固有の差分より汎用コードの方が大きくなる
- 汎用関数を `ralph` サブコマンドに分離すると、shell script は薄いグルーコードに過ぎなくなり、生成物としての存在意義が失われる
- Go runner が直接ループを駆動すれば、shell script 生成・テンプレート管理・generation-time と runtime の境界問題がすべて解消される

本設計では以下を採用する。
- 3-phase ループ（`pre` / `main` / `post`）
- GitHub Actions 風の `if` 条件式（Go 内部 evaluator）
- builtin 参照の最小化（`uses: auto_commit` のみ）

一方で、機能過剰になる以下は採用しない。
- profile 切り替え（minimal / standard など）
- shell script 生成モード

初期化 CLI は `config.yml` / `prompt.md` / `prd.json` / `progress.md` の雛形作成に限定して採用する。

## Design

### Proposed Approach

1. `.ralph/config.yml` を SoT（source of truth）として扱う。
2. `ralph run` が `.ralph/config.yml` を読み取り、Go プロセスとしてタスクループを直接実行する。
3. `ralph init` で `.ralph/config.yml`、`.ralph/prompt.md`、`.ralph/prd.json`、`.ralph/progress.md` の雛形を初回生成できるようにする。
4. main（prompt を coding agent に渡す処理）は最小要件として固定し、config で変更できない。
5. `phases.pre` と `phases.post` の step は同一仕様（`run` or `uses` / `if` / `on_fail`）を持てる。
6. builtin 参照は `uses: auto_commit` のみを認める（拡張は将来検討）。
7. 追加エージェント起動を伴う処理は scope 外とし、ローカルコマンドのみ runner に含める。
8. `ralph run --dry-run` で実行計画（step 一覧、`if` 式、`on_fail` 設定、agent command、git commit mode）を事前確認できるようにする。

### Architecture

#### Components

- **initializer（Go, `ralph init`）**
  - Template source: `go:embed` で内包した `config.yml` / `prompt.md` / `prd.json` / `progress.md` 向けテンプレート
  - `.ralph/` が存在しない場合は `os.MkdirAll(".ralph", 0o755)` 相当で先に作成する
  - `config.yml` / `prompt.md` / `prd.json` はテンプレートをそのままコピーし、`progress.md` のみ `text/template` 展開（`{{ .Today }}`）を行う
  - Output: `.ralph/config.yml`, `.ralph/prompt.md`, `.ralph/prd.json`, `.ralph/progress.md`
- **runner（Go, `ralph run`）**
  - Input: `.ralph/config.yml`, `.ralph/prompt.md`, `.ralph/prd.json`, `.ralph/progress.md`, `.ralph/.commit-msg`
  - Behavior: config.yml を読み取り、固定 main + configurable pre/post のタスクループを Go プロセスとして直接実行する
  - step の `run` コマンドは `os/exec` で `sh -c "<command>"` として子プロセス実行する
  - `if` 式は Go 内部の evaluator で評価する
  - `uses: auto_commit` は Go 内部で `git` コマンドを `os/exec` 経由で実行して処理する
  - `--dry-run` で実行計画を出力し、実際のコマンド実行を行わない
- **runtime prerequisites**
  - `ralph run` 実行環境は `git` を前提とする（`jq` は不要）

#### Data Flow (1 Iteration)

1. runner は loop 開始前に `.ralph/prd.json` を検証し、以下のいずれかを満たさない場合は設定エラー（exit code 22）として終了する。
   - `stories` が 1 件以上ある配列であること
   - 各 story の `id` が non-empty string であること
   - story `id` が `stories` 内で一意であること
   - 各 story の `passes` が boolean であること
   - 各 story の `deps` が string 配列であること
2. runner は iteration 開始時に `.ralph/prd.json` の内容をメモリ上に before snapshot として保持する。
3. runner が `phases.pre.steps` を順に実行する。各 step の `if` は Go 内部 evaluator で評価する。
4. runner が固定 main で `.ralph/prompt.md` を agent に渡して 1 タスク実行する。
   - main は `os/exec` で `sh -c "<agent.command>"` を起動し、`.ralph/prompt.md` の内容を stdin で渡す（prompt path を引数としては渡さない）。
   - agent の stdout は tmpfile（`os.CreateTemp`）へ保存し、stderr は親プロセスへ継承する。
   - main の成否は agent process の終了結果で判定し、exit code `0` のみ成功、non-zero 終了またはシグナル終了は失敗とする。
   - completion 判定には output 全文ではなく末尾 `completion.tail_lines` 行のみを利用する。
5. main（agent command）が `.commit-msg`、`prd.json`（passes 更新）、`progress.md` を更新する。
6. runner が `phases.post.steps`（fmt/lint/test/commit など）を `if` と `on_fail` ルールで実行する。
   - main が失敗した場合でも post phase は実行する（post phase 開始時は `success()=false`, `failure()=true`）。
7. `uses: auto_commit` step が有効な場合、`git.commit` に応じて commit する。
   - `split`: 第1 commit は `git add -A .ralph/` で `.ralph/` 配下のみを stage して作成する（untracked を含む）。
   - `split`: 第1 commit 用の staged 差分が空なら `.ralph/` commit は no-op で skip する（失敗にしない）。
   - `split`: 第1 commit を作成する場合のみ `.ralph/` commit message は `chore(ralph): mark ${task_id} complete in PRD and progress` を使う。
   - `split` の `${task_id}` は第1 commit を作成する場合のみ、before snapshot と現在の `.ralph/prd.json` を比較して `passes: false -> true` へ遷移した story ID から決定する。候補が 1 件のときのみ採用し、0 件または 2 件以上なら auto_commit を失敗として扱う。
   - `split`: 第2 commit は `git add -A` で全変更を stage した後、`git restore --staged .ralph/` で `.ralph/` を index から外して `.ralph/` 以外のみを commit する。
   - `split`: 第2 commit 用の staged 差分が空なら no-op で skip する（`.ralph/` のみ変更された iteration でも失敗させない）。
   - `split`: 第2 commit の message は `.ralph/.commit-msg` が空でなければファイル全体（複数行可）を使い、空なら `git.fallback_message`（デフォルト: `feat: implement task (auto-commit)`）を使う。
   - `together`: `git add -A` で変更全体（untracked を含む）を stage して 1 commit する。message は `.ralph/.commit-msg` が空でなければファイル全体（複数行可）を使い、空なら `git.fallback_message` を使う。`${task_id}` 抽出は行わない。
   - `together`: staged 差分が空なら no-op で skip する（失敗にしない）。
   - `.ralph/.commit-msg` の「空」は「ファイルが存在しない」または「`strings.TrimSpace(content) == \"\"`」を指す。非空時は trim 前のファイル全体を commit message として使う。
8. 完了判定を評価する。
   - main が成功した iteration のみ completion 判定を実行する。main が失敗した iteration では completion 判定をスキップし、未完了として扱う。
   - `completion.strategy=tail_match`: 「全 stories が `passes=true`」かつ「agent 出力末尾 `completion.tail_lines` 行のいずれか 1 行が `completion.signal` と完全一致」の両方を満たしたとき complete。
   - main が成功していて全 stories が `passes=true` だが signal が一致しない場合は protocol mismatch として失敗扱いにする。
9. 完了判定（または main 失敗時の判定スキップ）後、当該 iteration で作成した tmpfile を削除する。
10. complete なら終了し、未完了なら `agent.sleep_seconds` 待機後に次 iteration に進む。

### Config Schema (Draft, `.ralph/config.yml`)

```yaml
# Full schema example.
# `ralph init` で出力する `config.yml` はこのうち最小項目のみ。
# yaml-language-server: $schema=https://raw.githubusercontent.com/shuymn/ralph/main/schemas/config.schema.json
version: "1"

agent:
  command: claude -p --dangerously-skip-permissions
  max_iterations: 60
  sleep_seconds: 5

completion:
  strategy: tail_match
  signal: "<promise>COMPLETE</promise>"
  tail_lines: 20

git:
  # split: `.ralph/` とそれ以外を分けて commit
  # together: 変更をまとめて 1 commit
  commit: split
  fallback_message: "feat: implement task (auto-commit)"

phases:
  pre:
    steps:
      - name: pre_check
        run: bash scripts/pre-check.sh
        if: always()
        on_fail: stop_loop
  post:
    steps:
      - name: fmt
        run: cargo +nightly-2026-02-20 fmt --all
        if: always()
        on_fail: continue
      - name: clippy_fix
        run: cargo clippy --workspace --all-targets --fix --allow-dirty --allow-staged -- -D warnings
        if: always()
        on_fail: continue
      - name: tests
        run: cargo nextest run --workspace
        if: always()
        on_fail: stop_loop
      - name: auto_commit
        uses: auto_commit
        if: changed()
        on_fail: stop_loop
```

`ralph init` で出力する `config.yml` は最小構成とし、例は以下。

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/shuymn/ralph/main/schemas/config.schema.json
version: "1"

agent:
  command: "claude -p --dangerously-skip-permissions"

git:
  commit: split
  fallback_message: "feat: implement task (auto-commit)"

phases:
  pre:
    steps: []
  post:
    steps:
      - name: auto_commit
        uses: auto_commit
        if: changed()
        on_fail: stop_loop
```

v1 のデフォルト値は以下。

| Key | Default | Notes |
| --- | --- | --- |
| `agent.max_iterations` | `60` | `init` 最小出力では省略可能 |
| `agent.sleep_seconds` | `5` | iteration 間 sleep 秒数 |
| `completion.strategy` | `tail_match` | 完了判定方式 |
| `completion.signal` | `<promise>COMPLETE</promise>` | `tail_match` 用の一致文字列 |
| `completion.tail_lines` | `20` | 末尾判定の対象行数 |
| `git.commit` | `split` | commit grouping |
| `git.fallback_message` | `feat: implement task (auto-commit)` | `.commit-msg` 空時に使用 |
| `step.if` | `success()` | 未指定時 |
| `step.on_fail` | `stop_loop` | 未指定時 |

設計上の意図:

- main（prompt -> coding agent）は固定経路とし、設定対象にしない。
- `phases.pre.steps` と `phases.post.steps` は同一仕様で上から順に実行する。
- `if` は GitHub Actions 風の最小 expr として Go 内部 evaluator で評価する。
- step は `run` か `uses` のどちらか一方のみを持つ（排他）。
- `on_fail` は `continue` または `stop_loop` の 2 値のみを許可する。
- `on_fail` 未指定時は `stop_loop` をデフォルトとする。
- builtin は `uses: auto_commit` のみ。追加 builtin は v1 範囲外とする。
- `project` は config 項目として持たない（project 識別は `prd.json` 側で扱う）。
- `paths` は config 項目として持たず、`.ralph/prompt.md` / `.ralph/prd.json` / `.ralph/progress.md` / `.ralph/.commit-msg` に固定する。
- `git.commit` は `split` / `together` の 2 値のみとし、commit grouping 方針だけを設定する。
- `git.commit=split` の `.ralph/` commit message は `chore(ralph): mark ${task_id} complete in PRD and progress` で固定し、`${task_id}` は `prd.json` before/after 差分（`passes: false -> true`）から 1 件だけ抽出して決定する。
- `.ralph/` 以外の commit message は `.ralph/.commit-msg`（ファイル全体、複数行可）を優先し、空の場合は `git.fallback_message`（デフォルト: `feat: implement task (auto-commit)`）を使う。
- `.ralph/config.yml` の先頭非空行に `yaml-language-server` の `$schema` URL を埋め込み、JSON Schema をリモート参照する。

### Runner Execution Model

#### Loop Structure

`ralph run` は以下のループを Go プロセスとして直接実行する:

```
validate_prd_stories(".ralph/prd.json")  # else exit(22)
for iteration in 1..max_iterations:
    before_snapshot = read(".ralph/prd.json")
    if run_steps(pre_steps) == stop_loop:
        exit(20)
    main_result, main_output_tmp = run_main(agent_command, ".ralph/prompt.md")
    if run_steps(post_steps, main_result) == stop_loop:
        cleanup(main_output_tmp)
        exit(20)
    if main_result.success && check_completion(main_output_tmp):
        cleanup(main_output_tmp)
        exit(0)
    cleanup(main_output_tmp)
    sleep(sleep_seconds)
exit(23)  # max_iterations reached
```

main は固定（prompt を coding agent に渡す）とし、拡張点は pre/post のみとする。プロジェクト差分は pre/post steps と周辺設定から注入する。

#### Step Execution

step の `run` コマンドは `os/exec` で `sh -c "<command>"` として実行する。

- working directory は `ralph run` を実行したディレクトリ（プロジェクトルート）とする。
- stdout / stderr は親プロセス（`ralph run`）にそのまま継承する。
- 終了コード 0 を成功、non-zero を失敗として扱う。

`uses: auto_commit` は Go 内部で `git` コマンドを `os/exec` 経由で実行する。shell script への展開は行わない。

#### Main Phase Execution

`run_main` は `agent.command` を `os/exec` で `sh -c "<agent.command>"` として起動する。`cat .ralph/prompt.md | ...` のような shell pipeline 文字列は組み立てず、`.ralph/prompt.md` を開いて stdin pipe として直接接続する。

- working directory は `ralph run` 実行ディレクトリ（プロジェクトルート）とする。
- stdout は iteration ごとの tmpfile（`os.CreateTemp`）へ保存し、completion 判定入力として使う。
- stderr は親プロセス（`ralph run`）へ継承する。
- 成功条件は agent process の exit code が `0` であること。
- 失敗条件は agent process の non-zero 終了、シグナル終了、起動失敗（spawn error）であること。
- main 失敗時も loop は直ちに終了せず、post phase は実行する。completion 判定は main 成功 iteration のみ評価する。
- tmpfile は 1 iteration につき 1 つ作成し、当該 iteration の completion 判定後に削除する。`stop_loop` や異常終了（exit `20/21/22/23`）、SIGINT/SIGTERM 受信時でも、作成済み tmpfile は終了前に best-effort で削除する。

#### Step Condition Expression (`if`)

`if` は GitHub Actions 記法を参考にした最小 expr とする。shell 式は受け付けない。Go 内部の evaluator で直接評価する。

- 例:
  - `if: always()`
  - `if: changed()`
  - `if: success() && changed()`
- サポート範囲（v1）:
  - リテラル: `true`, `false`
  - 関数: `always()`, `success()`, `failure()`, `changed()`
  - 演算子: `!`, `&&`, `||`, `(`, `)`
  - 非対応: `steps.<name>.success` など step 名参照
- デフォルト:
  - `if` 未指定は `success()` 相当で評価する
- `if` 式は config load 時（`ralph run` と `ralph run --dry-run` の両方）に parse/validation を行い、runtime では評価のみを行う。非対応構文や parse error は実行前に exit code `22` とする。

`changed()` は `git status --porcelain` の標準出力が非空かどうかで判定する（exit code ではなく出力文字列で判定する）。対象範囲は working tree 全体（untracked files を含む）であり、`.ralph/` 配下の変更も含む。評価は phase 開始時にキャッシュせず、各 step の `if` 評価時点で毎回再評価する。`git status --porcelain` の実行自体が失敗した場合は `if` expression error として扱い、runner は exit code `22` で終了する。

例: post phase の step A が `cargo fmt` でファイルを変更した場合、続く step B の `if: changed()` は true と評価される。

`success()` / `failure()` の評価スコープは phase 単位で定義する。

- pre phase 開始時: `success()=true`, `failure()=false`
- post phase 開始時:
  - main が成功: `success()=true`, `failure()=false`
  - main が失敗: `success()=false`, `failure()=true`
- phase 内で step が失敗した場合:
  - `on_fail=continue` なら次 step 以降は `success()=false`, `failure()=true`
  - `on_fail=stop_loop` ならその時点で loop を終了する
- pre phase で `on_fail=stop_loop` が発生した場合は、その iteration の main/post は実行しない。

#### Step Schema (Run vs Uses)

各 step は以下のいずれか一方を持つ。

- `name: <string>`: 必須。空文字・空白のみは不可。同一 phase 内で重複不可。
- `run: <command>`: 任意コマンドを `sh -c` で実行
- `uses: auto_commit`: builtin を呼び出し

`run` と `uses` の同時指定、および両方未指定は設定エラーとする。
`name` の欠落・空値・重複は設定エラーとする。

`on_fail` は `continue` または `stop_loop` を受け付け、未指定時は `stop_loop` を適用する。

### Command Semantics

- `ralph init`: `.ralph/config.yml`、`.ralph/prompt.md`、`.ralph/prd.json`、`.ralph/progress.md` の雛形を初期化時に生成する。
- `ralph init` は `.ralph/` が存在しない場合にディレクトリを先に作成してから各雛形を書き込む。
- `ralph init` は既存ファイルを上書きしない。既存ファイルは skip し、標準エラー出力に skip 一覧を表示する。
- `ralph init --force` は v1 では提供しない。
- `ralph init` の `config.yml` は最小項目のみを出力し、未出力項目は runner のデフォルト値を使う。
- `ralph init` の `config.yml` には `git.commit`（`split` / `together`）と `git.fallback_message` を含める。
- `git.fallback_message` のデフォルトは `feat: implement task (auto-commit)` とする。
- `config.version` は v1 では JSON/YAML string の `"1"` のみを受け付ける。未指定、未対応 version（例: `"2"`）、型不一致（例: `1`）は validation error（exit code 22）とする。
- `ralph init` が出力する `.ralph/config.yml` の先頭（先頭非空行）には、以下の schema directive comment を含める。
  - `# yaml-language-server: $schema=https://raw.githubusercontent.com/shuymn/ralph/main/schemas/config.schema.json`
- schema は GitHub（raw URL）から参照し、`.ralph/` 配下に schema 実ファイルは生成しない。schema 本体（`schemas/config.schema.json`）はリポジトリ管理物として Phase 0 で作成・公開する。
- init テンプレートは `go:embed` でバイナリに内包されたものを使用する。
- `config.yml` / `prompt.md` / `prd.json` はテンプレートをそのままコピーし、`progress.md` のみ `text/template` で展開して `Started:` に当日の日付（`YYYY-MM-DD`）を埋め込む。
- `ralph run`: `.ralph/config.yml` を読み取り、タスクループを Go プロセスとして直接実行する。
- `ralph run` は loop 開始前に `.ralph/prd.json` を検証し、`stories` 非空・story `id` 非空/一意・`passes` boolean・`deps` string 配列を満たさない場合は設定エラーとして `exit 22` で終了する（pre/main/post は実行しない）。
- fixed main は `sh -c "<agent.command>"` で起動し、`.ralph/prompt.md` は stdin で渡す（引数で prompt path は渡さない）。
- fixed main の成功は exit code `0`、失敗は non-zero 終了・シグナル終了・起動失敗として扱う。
- main が失敗しても post phase は実行される。completion 判定は main 成功 iteration だけで実行し、main 失敗 iteration は未完了として次 iteration へ進む（post で `stop_loop` が発生した場合は exit 20）。
- `git.commit=together` では `${task_id}` 抽出を行わない。commit message は `.ralph/.commit-msg` または `git.fallback_message` のみで決定する。
- `git.commit=split` では `git add -A .ralph/`（第1 commit）と `git add -A` + `git restore --staged .ralph/`（第2 commit）で staging を分離し、各 commit の staged 差分が空なら no-op で skip する。
- auto_commit の staging は `git add -A` 相当を前提とし、untracked files を含む。`git add -A` が取り込むファイルサイズ/種別への追加ガードは v1 では提供しない（`.gitignore` と運用で制御する）。
- auto_commit は最終的な staged 差分が空の場合に no-op success として扱い、`git commit` エラーにしない。
- `.ralph/.commit-msg` の「空」は「ファイル欠如」または「trim 後空文字列」で判定する。非空時は trim 前のファイル全体（複数行可）を commit message として使う。
- `ralph run --dry-run`: 実行計画を標準出力へ出力し、実際のコマンド実行は行わない。出力には以下を含む:
  - agent command と max_iterations / sleep_seconds
  - pre/post steps の一覧（各 step の `name`, `run` or `uses`, `if`, `on_fail`）
  - git commit mode と fallback message
  - completion strategy / signal / tail_lines
- `ralph run --dry-run` は `if` 式と config/prd の validation も実行し、実行前に parse/validation error を検出する。
- runner が参照するファイル path（`prompt` / `prd` / `progress` / `.commit-msg`）は `.ralph/` 配下の固定 path を使用し、config からは変更できない。
- runner が `prd.json` で必須として参照するのは `stories`（各 story の `id` / `passes` / `deps` を含む）である。`id` は non-empty かつ一意、`passes` は boolean、`deps` は `[]string` として検証する。dependency graph（循環、未定義参照、実行順）は runner では検証・利用しない。
- `deps` に基づくタスク選択/ブロック判定は prompt に従う agent 側の責務とし、runner の completion 判定には使わない。
- `project` / `plan` とその他の top-level fields は runner では参照せず許容する（agent/human 向けメタデータとして保持可能）。
- step の `run` コマンドは `os/exec` で `sh -c "<command>"` として実行する。working directory は `ralph run` を実行したディレクトリとする。
- step の `if` 式は Go 内部 evaluator で評価し、構文/サポート範囲の validation は config load 時に完了させる。
- step の `on_fail` 未指定時は `stop_loop` として扱う。
- step の `name` は必須で、空値または同一 phase 内重複は config validation error（exit code 22）とする。
- `uses: auto_commit` は Go 内部で `git` コマンドを `os/exec` 経由で実行する。
- `git.commit=split` では `.ralph/` とそれ以外を別 commit に分離し、`.ralph/` commit を作成する場合のみ `${task_id}` を `prd.json` の before/after snapshot 差分（`passes: false -> true`）から 1 件だけ抽出して決定する。
- before snapshot はメモリ上に保持する（tmpfile や `.ralph/` 配下へのファイル書き出しは行わない）。
- `.commit-msg` が非空の場合（空判定は「ファイル欠如または trim 後空文字列」）、commit message はファイル全体（subject + body）を使用する。
- fixed main の agent 出力（stdout）は tmpfile（`os.CreateTemp`）へ保存し、completion 判定は tmpfile の tail のみを参照する。`tail_match` の signal 判定は「tail 内の 1 行が `completion.signal` と完全一致したか」で行う。tmpfile は iteration ごとに削除する。
- `completion.signal` をデフォルトから変更した場合は、`.ralph/prompt.md` の Stop Condition literal も同じ文字列へ合わせる。`ralph run --dry-run` 出力に `completion.signal` を含め、実行前にズレを検知できるようにする。
- `phases.main` は設定項目として提供しない（固定 main のため）。

### Runtime Status and Exit Codes

- 本節の `20-23` は `ralph run` 専用の終了コードとする。
- `0`: 正常完了（complete）
- `20`: `on_fail=stop_loop` により loop 中断（step 失敗）
- `21`: completion protocol mismatch（main 成功 iteration で全 stories `passes=true` だが、`completion.signal` と完全一致する行が tail で未検出）
- `22`: runtime 評価エラー（`if` expression error、config/prd validation error など。例: `stories: []`, `changed()` 内部の `git status` 実行失敗）
- `23`: `max_iterations` 到達で未完了
- `stop_loop` 発生時は、`[ralph] stop_loop phase=<phase> step=<step> reason=<reason>` を標準エラーへ出力する
- main 失敗単体では即時終了しない。post を経由し、`stop_loop` がなければ当該 iteration の completion 判定はスキップして次 iteration へ進むため、main 失敗 iteration 単体で `21` にはならない。
- `ralph init` は通常の CLI と同様に `0=success`, `non-zero=error` を使用する。

### Init Scaffold: `progress.md`

`ralph init` で生成する `.ralph/progress.md` の雛形は以下をデフォルトとする。

```md
# Ralph Progress Log
Started: {{ .Today }}

## Codebase Patterns
- General: Keep changes small

## Key Files
- README.md
---
```

`Today` は `ralph init` 実行日のローカル日付（`YYYY-MM-DD`）を generator 側で渡す。

### Safety and Risks

1. step command injection: `config.yml` の `run` に悪意あるコマンドが入ると任意実行される。
Mitigation: `config.yml` はリポジトリ内に管理し、PR レビューで検証する。`ralph run` は `config.yml` を信頼する前提とする。
2. quality gate の形骸化: `on_fail: continue` の濫用で品質低下する。
Mitigation: `tests` は原則 `on_fail: stop_loop` を運用規約にする。
3. N+1 起動混入: post step に追加エージェント呼び出しを入れると起動回数が増える。
Mitigation: step はローカルコマンド限定、追加起動ロジックは設計上禁止する。
4. 雛形の意図せぬ上書き: 運用中の `config.yml` / `prompt.md` / `prd.json` / `progress.md` が失われるリスクがある。
Mitigation: `ralph init` は既存ファイルを上書きせず skip する。
5. schema URL ドリフト: schema path 変更時に既存 `config.yml` の補完が壊れる。
Mitigation: schema URL を定数として一元管理し、変更時は互換リダイレクトまたは告知を行う。
6. `task_id` 決定の曖昧さ: 1 iteration で複数 story が完了すると tracking commit が壊れる。
Mitigation: `task_id` は before/after snapshot から `passes: false -> true` に遷移した story を抽出し、1 件のみ許可する（0 件または 2 件以上は auto_commit 失敗）。
7. completion protocol mismatch: main 成功 iteration で stories は完了していても signal 不一致だと完了判定が成立しない。
Mitigation: `tail_match` では signal 不一致を明示エラー（exit code 21）として扱い、runner が理由を標準エラーへ出力する。
8. 空 `stories` の見落とし: `stories: []` を誤って投入すると、実行意図と異なる完了判定になるリスクがある。
Mitigation: loop 開始前に `stories` 非空を検証し、空配列は設定エラー（exit code 22）として拒否する。
9. tmpfile のリーク: iteration ごとの agent 出力 tmpfile が残留するとディスクを圧迫する。
Mitigation: completion 判定後に毎 iteration 削除し、exit `20/21/22/23` 経路および SIGINT/SIGTERM 受信時でも作成済み tmpfile を best-effort で削除する。
10. completion signal の不整合: `completion.signal` だけ変更して `.ralph/prompt.md` の Stop Condition literal を更新しないと complete 判定が成立しない。
Mitigation: `completion.signal` を変更する場合は prompt 側 literal も同時に更新する。`ralph run --dry-run` に signal を表示し、実行前確認を必須運用にする。
11. auto_commit の過剰 staging: `git add -A` により意図しない大容量ファイルや補助生成物が commit 対象に入るリスクがある。
Mitigation: v1 では file-size/type ガードを持たない。`.gitignore` とレビュー運用で制御する。
12. `deps` グラフ不整合: `deps` に循環や未定義参照があっても runner が検出しないため、agent のタスク選択が停滞するリスクがある。
Mitigation: `deps` の意味論チェック（循環・参照整合）は agent/human の運用責務とし、必要なら将来バリデータを検討する。
13. agent プロセスのハング: agent command が応答しなくなると iteration が停止する。
Mitigation: v1 では OS レベルの timeout（`timeout` コマンドを `agent.command` に組み込む等）で対応する。runner 側の timeout 機構は将来検討とする。

### Testing Strategy

#### Phase-to-Test Mapping (Done Gate)

| Phase | Done 条件 | 対応テスト |
| --- | --- | --- |
| Phase 0（基盤） | config decode/validation・`if` evaluator・schema artifact の仕様が固定される | `config`, `if evaluator` の全項目が green |
| Phase 1（初期化） | `ralph init` の雛形生成・非上書き・schema directive 出力が仕様どおり動く | `init` の全項目が green |
| Phase 2（runner） | `ralph run` の core loop（pre/main/post, completion, auto_commit）が仕様どおり動く | `runner` の Phase 2 gate 項目が green |
| Phase 3（安定化） | runtime 全体の運用挙動（exit code, dry-run, sleep, fixture 回帰）が安定する | `runner` の Phase 3 gate 項目が green |

- config:
  - YAML decode 単体テスト（必須フィールド、デフォルト値）
  - `version` 検証テスト（string `"1"` を受理し、未指定・未対応 version・型不一致 `1` は設定エラー（exit code 22）にすること）
  - step `name` 検証テスト（必須・空値禁止・同一 phase 内重複禁止）
  - step schema 検証テスト（`run` xor `uses`、`uses` の許可値チェック）
  - `on_fail` デフォルト評価テスト（未指定時に `stop_loop` が適用されること）
  - `git.commit` 値検証テスト（`split` / `together` 以外を設定エラーにすること）
  - 固定 path テスト（config に `paths` を持たず、runner が `.ralph/` 固定 path を参照すること）
  - schema artifact テスト（`schemas/config.schema.json` が存在し、`config.template.yml` の `$schema` URL と整合すること）
- if evaluator:
  - expr evaluator テスト（`always/success/failure/changed`, `!`, `&&`, `||`, 括弧）
  - `if` 省略時デフォルト評価テスト（`success()` 相当として動作すること）
  - `if` 非対応構文テスト（`steps.<name>.success` を実行前 validation error（exit code 22）にすること）
  - `if` validation タイミングテスト（`ralph run` と `ralph run --dry-run` の両方で config load 時に parse/validation を行うこと）
  - `success()` / `failure()` スコープテスト（pre 初期値、post の main 成否反映、`on_fail=continue` 後の反転）
  - `changed()` 判定スコープテスト（untracked のみ変更時、および `.ralph/` のみ変更時でも true になること）
  - `changed()` 再評価タイミングテスト（phase 開始時キャッシュではなく、各 step の `if` 評価時に再度 `git status --porcelain` で判定すること）
  - `changed()` 実行失敗テスト（`git status --porcelain` が失敗した場合、`if` expression error として exit code 22 になること）
- runner:
  - [Phase 2 gate] step execution テスト（`run` コマンドが `sh -c` で実行されること）
  - [Phase 2 gate] pre/post step 順序テスト（steps が定義順に実行されること）
  - [Phase 2 gate] pre `stop_loop` テスト（pre で `on_fail=stop_loop` 発生時に main/post を実行せず exit code 20 で終了すること）
  - [Phase 2 gate] main phase 実行テスト（`sh -c "<agent.command>"` で起動し、`.ralph/prompt.md` を stdin で渡すこと）
  - [Phase 2 gate] main 成否判定テスト（exit code `0` を成功、non-zero/シグナル終了/起動失敗を失敗として扱うこと）
  - [Phase 2 gate] main 失敗時フローテスト（post phase を実行し、post 開始時 `success()=false`, `failure()=true` となり、当該 iteration の completion 判定はスキップされること）
  - [Phase 2 gate] agent output 保持テスト（出力を tmpfile に保存し、completion は tail のみを参照すること）
  - [Phase 2 gate] before snapshot テスト（iteration 開始時にメモリ上に保持すること）
  - [Phase 2 gate] `stories` 検証テスト（`stories: []`、`id` 空文字、`id` 重複、`passes` 非 bool、`deps` 非 `[]string` を loop 開始前に exit code 22 で拒否すること）
  - [Phase 2 gate] `task_id` 抽出テスト（before/after diff から `passes: false -> true` を検出し、`0/1/2+` 件を判定できること）
  - [Phase 2 gate] `git.commit=split` commit 分離テスト（`.ralph/` と非`.ralph/` が別 commit になること）
  - [Phase 2 gate] `git.commit=split` staging テスト（第1 commit が `git add -A .ralph/` 相当、第2 commit が `git add -A` + `git restore --staged .ralph/` 相当であること）
  - [Phase 2 gate] `git.commit=split` 空 commit 回避テスト（第1/第2 commit それぞれ staged 差分が空なら no-op skip し、`git commit` 失敗にならないこと）
  - [Phase 2 gate] `git.commit=together` テスト（`${task_id}` 抽出を行わず、`.commit-msg` / `git.fallback_message` のみで commit message を決定すること）
  - [Phase 2 gate] `git.commit=together` no-changes テスト（staged 差分が空なら no-op success になること）
  - [Phase 2 gate] untracked staging テスト（split/together ともに `git add -A` 相当で untracked files を取り込むこと）
  - [Phase 2 gate] `.ralph/` commit message テスト（`chore(ralph): mark ${task_id} complete in PRD and progress` 形式になること）
  - [Phase 2 gate] `.commit-msg` 空判定テスト（ファイル欠如・空文字・whitespace-only は空扱い、非空はファイル全体を使用すること）
  - [Phase 2 gate] 非`.ralph/` message デフォルトテスト（`.ralph/.commit-msg` 空時に `git.fallback_message` を使うこと）
  - [Phase 2 gate] 非`.ralph/` message 優先テスト（`.ralph/.commit-msg` 非空時にファイル全体を commit message として使うこと）
  - [Phase 2 gate] completion 判定テスト（`tail_match` で `passes=true` かつ signal と完全一致する行が tail にあるときのみ complete、substring のみ一致は不一致として扱うこと）
  - [Phase 2 gate] protocol mismatch テスト（main 成功 iteration で `passes=true` かつ signal 不一致のときのみ exit code 21 になること）
  - [Phase 2 gate] `prd.json` permissive field テスト（`project` / `plan` / unknown top-level fields が存在しても、`stories` 要件を満たせば runner が動作すること）
  - [Phase 2 gate] `deps` 方針テスト（`deps` は `[]string` 型を検証し、循環/未定義参照は runner で検証しないこと）
  - [Phase 3 gate] exit code テスト（`0/20/21/22/23` を期待通り返すこと）
  - [Phase 3 gate] `--dry-run` テスト（実行計画が出力され、コマンドが実行されないこと）
  - [Phase 3 gate] sleep テスト（iteration 間で `agent.sleep_seconds` 待機すること）
  - [Phase 3 gate] tmpfile lifecycle テスト（iteration 正常完了/継続時、exit `20/21/22/23` 経路、および SIGINT/SIGTERM 受信時に作成済み tmpfile が削除されること）
  - [Phase 3 gate] fixture 回帰テスト（代表的な config/prd/prompt で `run`/`--dry-run` の期待挙動を再現できること）
- init:
  - `init` ディレクトリ作成テスト（`.ralph/` 未存在時に先に作成されること）
  - `init` テスト（`config.yml` / `prompt.md` / `prd.json` / `progress.md` が未存在時のみ生成されること）
  - `init` 再実行テスト（既存ファイルを上書きせず skip 一覧を出力すること）
  - `init` 最小出力テスト（`config.yml` が最小項目 + `git.commit` + `git.fallback_message` を出力し、欠落項目にデフォルトが適用されること）
  - schema directive 出力テスト（`config.yml` 先頭非空行に `yaml-language-server` の `$schema` comment が含まれること）
  - 非展開テスト（`init` 実行時に `.ralph/config.schema.json` など schema 実ファイルを作成しないこと）
  - テンプレート展開範囲テスト（`prompt.md` / `prd.json` は verbatim copy、`progress.md` のみ `{{ .Today }}` が展開されること）
  - `progress.md` 日付埋め込みテスト（`Started:` が init 実行日の `YYYY-MM-DD` になること）

### Rollout / Migration

1. **Phase 0（基盤）**: Go プロジェクト初期化、config YAML decode、`if` evaluator、`schemas/config.schema.json` を実装する。完了条件は `Testing Strategy` の `config` と `if evaluator` の全項目が green であること。
2. **Phase 1（初期化）**: `ralph init` で `config.yml` / `prompt.md` / `prd.json` / `progress.md` 雛形の初回生成を実装する。完了条件は `Testing Strategy` の `init` 全項目が green であること。
3. **Phase 2（runner）**: `ralph run` で pre/main/post ループ、`uses: auto_commit`、completion check を実装する。完了条件は `Testing Strategy` の `runner` にある `[Phase 2 gate]` 項目がすべて green であること。
4. **Phase 3（安定化）**: fixture テストを揃え、`--dry-run` を実装する。完了条件は `Testing Strategy` の `runner` にある `[Phase 3 gate]` 項目がすべて green であること。

### Alternatives Considered

- **Option A: 手編集で `ralph.sh` を育てる**
  - 利点: 最短で動く。
  - 欠点: プロジェクト増加時にコピペ分岐管理が破綻しやすい。
- **Option B: YAML + template で `ralph.sh` 生成**
  - 利点: 生成物が human-readable な shell script でレビュー可能。
  - 欠点: template に汎用 runtime 関数が肥大化し、プロジェクト固有の差分が埋没する。runtime helper サブコマンドを増やすと shell script が薄いグルーに過ぎなくなり、生成物の存在意義が失われる。generation-time と runtime の境界問題が設計を複雑化させる。
- **Option C: Go runner が config を直接読み取り実行（採用）**
  - 利点: shell script 生成不要。template 管理不要。`jq` 依存不要。`if` 式評価や JSON 処理を Go 型安全に実装できる。`--dry-run` で実行計画を事前確認できる。
  - 欠点: 「何が実行されるか」が config.yml と Go コードに分散する（`--dry-run` で緩和）。
- **Option D: profile や複数生成物を含むフル機能 generator**
  - 利点: 将来拡張性は高い。
  - 欠点: 現時点の要件に対して機能過剰。

### Concrete Examples

#### Example 1: post step に eslint を追加

```yaml
phases:
  post:
    steps:
      - name: fmt_rust
        run: cargo fmt --all
        if: always()
        on_fail: continue
      - name: lint_js
        run: npm run lint
        if: always()
        on_fail: stop_loop
      - name: tests
        run: cargo nextest run --workspace
        if: always()
        on_fail: stop_loop
```

#### Example 2: 複雑な step をスクリプトへ分離

```yaml
phases:
  post:
    steps:
      - name: project_checks
        run: bash scripts/ci-check.sh
        if: always()
        on_fail: stop_loop
```

#### Example 3: `if` で変更時のみ auto commit

```yaml
phases:
  post:
    steps:
      - name: auto_commit
        uses: auto_commit
        if: changed()
        on_fail: stop_loop
```

`if` は Go 内部 evaluator で評価する。

#### Example 4: `prd.json` の最小互換

```json
{
  "project": "stateql-v1",
  "plan": "docs/plans/2026-02-21-stateql-v1-plan.md",
  "stories": [
    { "id": "task-0", "title": "Remove Bootstrap Placeholders", "deps": [], "passes": true },
    { "id": "task-1", "title": "Add Stage-Typed Error Enums", "deps": ["task-0"], "passes": false }
  ]
}
```

## Decision Log

| ADR | Decision | Status |
| --- | --- | --- |
| [0001](../adr/0001-go-runner-over-shell-generation.md) | Go runner が config.yml を直接読み取りタスクループを実行する（shell script 生成を廃止） | Accepted |
| - | `if` は Go 内部 evaluator で評価し、`if` 未指定時は `success()` 相当、`steps.<name>.success` は v1 非対応とする | Accepted |
| - | `config.yml` は GitHub raw URL の JSON Schema を参照する。schema 本体はリポジトリ管理物として提供し、`ralph init` は `.ralph/` 配下へ schema 実ファイルを生成しない | Accepted |
| - | `project` / `paths` は config から外し、`git.commit` は `split` / `together` の 2 値に限定する | Accepted |
| - | `git.commit=split` では `.ralph/` を先に commit し、`${task_id}` は `.ralph/` commit を作成する場合のみ `passes: false -> true` 差分から 1 件抽出する。各 commit の staged 差分が空なら no-op で skip する | Accepted |
| - | `git.commit=together` では `${task_id}` 抽出を行わず、`.commit-msg` / `fallback_message` のみで commit message を決定する | Accepted |
| - | auto_commit の staging は split/together ともに `git add -A` 系を用い、untracked files を含む。v1 は file-size/type ガードを持たない | Accepted |
| - | step `name` は必須（空値不可）かつ同一 phase 内で一意とし、違反は config validation error（exit code 22）とする | Accepted |
| - | `.ralph/.commit-msg` の空判定は「ファイル欠如または trim 後空文字列」とし、非空時はファイル全体を commit message に使う | Accepted |
| - | `completion.strategy=tail_match` は main 成功 iteration でのみ評価し、「全 `passes=true`」かつ「tail 内の 1 行が `completion.signal` と完全一致」の AND で判定する | Accepted |
| - | `prd.json` の `stories` は 1 件以上必須で、各 story の `id` 非空/一意、`passes` bool、`deps` `[]string` を満たさない場合は設定エラー（exit code 22）として loop 実行前に失敗させる | Accepted |
| - | `config.version` は string `"1"` のみ受理し、未指定・未対応 version・型不一致（例: `1`）は設定エラー（exit code 22）とする | Accepted |
| - | runner は `deps` を `[]string` 型としてのみ検証し、dependency graph（循環/順序/未定義参照）は検証しない。タスク選択は agent 側責務とする | Accepted |
| - | runner は `prd.json` の `stories` を参照し、`project` / `plan` / unknown top-level fields は参照せず許容する | Accepted |
| - | fixed main は `sh -c "<agent.command>"` で起動し、`.ralph/prompt.md` を stdin で渡す（prompt path は引数で渡さない） | Accepted |
| - | main は exit code `0` のみ成功とし、non-zero/シグナル終了/起動失敗を失敗とする。main 失敗時も post は実行するが、completion 判定はスキップする | Accepted |
| - | agent output tmpfile は iteration ごとに作成し、completion 判定後に削除する（exit `20/21/22/23` と SIGINT/SIGTERM を含む終了経路で best-effort cleanup） | Accepted |
| - | `completion.signal` を変更した場合は `.ralph/prompt.md` 側の Stop Condition literal も同期して更新する（`--dry-run` で signal を確認できる） | Accepted |
| - | `ralph init` は既存ファイルを上書きせず skip し、`--force` は v1 で提供しない | Accepted |
| - | `ralph run` の終了コードを `0/20/21/22/23` へ固定する | Accepted |
| - | `changed()` は working tree 全体（untracked と `.ralph/` 変更を含む）を対象にし、phase 開始時に固定せず各 step の `if` 評価時に再評価する | Accepted |
| - | `changed()` 評価時に `git status --porcelain` が失敗した場合は `if` expression error とし、exit code 22 で失敗させる | Accepted |
| - | pre phase で `on_fail=stop_loop` が発生した場合は当該 iteration の main/post を実行しない | Accepted |
| - | before snapshot はメモリ上に保持する（tmpfile や `.ralph/` 配下へのファイル書き出しは行わない） | Accepted |
| - | `ralph run --dry-run` を v1 で提供する | Accepted |

## Open Questions

- 現時点ではなし。

## Acceptance Criteria

1. `ralph run` が `.ralph/config.yml` を読み取り、タスクループを Go プロセスとして直接実行できる。
2. タスクループは固定 main（prompt -> coding agent）と configurable pre/post で実行される。
3. `ralph init` で `.ralph/` を作成し、`.ralph/config.yml`、`.ralph/prompt.md`、`.ralph/prd.json`、`.ralph/progress.md` の雛形を生成できる。
4. `ralph init` は既存ファイルを上書きせず skip し、skip 対象を標準エラーへ出力する。
5. `ralph init` が出力する `.ralph/config.yml` は最小項目（`version`, `agent.command`, `git.commit`, `git.fallback_message`, `phases.pre`, `phases.post`）のみを含み、未指定項目にはデフォルトが適用される。
6. デフォルト値（`agent.max_iterations=60`, `agent.sleep_seconds=5`, `completion.strategy=tail_match`, `completion.signal`, `completion.tail_lines=20`, `step.if=success()`, `step.on_fail=stop_loop`）が仕様として定義される。
7. `ralph init` が出力する `.ralph/config.yml` の先頭非空行には `yaml-language-server` の `$schema` comment が含まれ、`https://raw.githubusercontent.com/shuymn/ralph/main/schemas/config.schema.json` を参照する。
8. `ralph init` は `.ralph/config.schema.json` など schema 実ファイルを展開せず、schema 本体はリポジトリ管理物（`schemas/config.schema.json`）として提供される。
9. `project` は `config.yml` の設定項目に存在しない。
10. `paths` は `config.yml` の設定項目に存在せず、runner の参照 path は `.ralph/` 配下固定である。
11. `git.commit` は `split`（`.ralph` とそれ以外を分割 commit）または `together`（1 commit）だけを受け付ける。
12. `git.commit=split` では `.ralph/` を先に commit し、message は `chore(ralph): mark ${task_id} complete in PRD and progress` を使う（`.ralph/` staged 差分が空なら no-op skip する）。
13. `git.commit=split` の `${task_id}` は `.ralph/` commit を作成する場合のみ `prd.json` before/after 差分で `passes: false -> true` になった story から 1 件のみ抽出して決定し、0 件または 2 件以上は auto_commit 失敗とする。
14. 非`.ralph/` の commit message は `.ralph/.commit-msg` 優先とし、非空時はファイル全体（複数行可）を使用する。空なら `git.fallback_message`（デフォルト: `feat: implement task (auto-commit)`）を使う。
15. `.ralph/prompt.md` は固定 1 枚で運用される。
16. `.ralph/progress.md` の `Started:` は `ralph init` 実行日の `YYYY-MM-DD` で埋め込まれる。
17. `if` 式は Go 内部 evaluator で評価され、`ralph run` / `ralph run --dry-run` の config load 時に parse/validation が完了する。
18. `if` 未指定時は `success()` 相当として評価される。
19. `if` に `steps.<name>.success` など step 名参照は使えない（v1 非対応）。
20. `success()` / `failure()` は phase スコープで評価される（pre 初期成功、post は main 成否を初期値に反映）。
21. builtin step 参照は `uses: auto_commit` のみである。
22. `on_fail` は `continue` / `stop_loop` のみを許可し、未指定時は `stop_loop` が適用される。
23. pre/post step はローカルコマンドのみ許可し、追加エージェント起動を含まない。
24. `completion.strategy=tail_match` は main 成功 iteration でのみ評価し、「全 stories `passes=true`」かつ「tail 内の 1 行が `completion.signal` と完全一致」の両方を満たすときのみ complete とし、不一致は protocol mismatch（exit code 21）とする。
25. `prd.json` の `stories` は 1 件以上必須であり、`stories: []`、story `id` 空/重複、`passes` 非 bool、`deps` 非 `[]string` の場合は `ralph run` は loop 開始前に exit code 22 で終了する（pre/main/post は実行しない）。
26. before snapshot はメモリ上に保持する。
27. `changed()` は working tree 全体（untracked files を含む）を対象にし、`.ralph/` 配下の変更も含めて判定する。phase 開始時に固定せず、各 step の `if` 評価時に再評価される。
28. `ralph run` の終了コード `20/21/22/23` は runner 専用とし、`stop_loop` 時は `phase` / `step` / `reason` を標準エラーへ出力する。
29. `ralph init` は `0=success`, `non-zero=error` の通常 CLI 終了コードを使用する。
30. `ralph run --dry-run` で実行計画（step 一覧、`if` 式、`on_fail`、agent command、git commit mode、completion 設定）が出力され、実際のコマンド実行は行われない。`if` 式と config/prd validation も実行前に検証する。
31. step の `run` コマンドは `sh -c` で実行され、working directory は `ralph run` を実行したディレクトリである。
32. `ralph run` の runtime 依存は `git` のみである（`jq` は不要）。
33. `prd.json` は `deps` / `passes` 前提の最小互換で動作する。
34. profile などの機能過剰要素が v1 スコープ外であることが明記されている。
35. shell script 生成（`ralph generate`）は v1 で提供されない。
36. fixed main は `sh -c "<agent.command>"` で起動され、`.ralph/prompt.md` は stdin で渡される（prompt path を引数としては渡さない）。
37. main は exit code `0` のみ成功とし、non-zero 終了・シグナル終了・起動失敗は失敗として扱われる。
38. main が失敗しても post phase は実行され、post 開始時の評価コンテキストは `success()=false`, `failure()=true` になる。
39. main 失敗後も post phase は実行されるが、当該 iteration の completion 判定はスキップされる。`stop_loop` がなければ未完了として次 iteration へ進む。
40. agent output tmpfile は iteration ごとに作成され、completion 判定後に削除される。exit `20/21/22/23` と SIGINT/SIGTERM の終了経路でも作成済み tmpfile は best-effort で削除される。
41. `git.commit=together` では `${task_id}` 抽出を行わず、commit message は `.ralph/.commit-msg` または `git.fallback_message` のみで決定される。
42. `config.version` は string `"1"` のみ受理し、未指定・未対応 version・型不一致（例: `1`）は validation error（exit code 22）として扱われる。
43. auto_commit の staging は split/together ともに `git add -A` 相当を用い、untracked files を含む。v1 では file-size/type ガードを提供せず、staged 差分が空の場合は no-op success とする。
44. runner は `deps` を `[]string` 型としてのみ検証し、dependency graph（循環/順序/未定義参照）は検証しない。`deps` に基づくタスク選択は agent 側責務である。
45. runner は `prd.json` の `stories`（`id` / `passes` / `deps`）を必須参照し、`id` 非空/一意・`passes` bool・`deps` `[]string` を検証する。`project` / `plan` と unknown top-level fields は許容する。
46. `completion.signal` を変更した場合、`.ralph/prompt.md` の Stop Condition literal も同じ文字列へ更新する。`--dry-run` 出力で signal 設定を確認できる。
47. step `name` は必須で、空値不可・同一 phase 内重複不可とし、違反は config validation error（exit code 22）として扱われる。
48. `.ralph/.commit-msg` の空判定は「ファイル欠如または trim 後空文字列」とし、非空時は trim 前のファイル全体を commit message として使用する。
49. `changed()` 評価時に `git status --porcelain` の実行が失敗した場合は `if` expression error とし、runner は exit code `22` で終了する。
50. `ralph init` のテンプレート適用は `config.yml` / `prompt.md` / `prd.json` が verbatim copy、`progress.md` のみ `text/template` 展開である。
51. pre phase で `on_fail=stop_loop` が発生した場合、当該 iteration の main/post は実行されない。
