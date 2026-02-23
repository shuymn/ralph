# 0001: Go Runner over Shell Script Generation

## Status

Accepted

## Context

ralph はプロジェクトごとの差分（lint/fmt/test 構成、commit 方針、完了判定）を YAML に集約し、タスクループを自動実行するツールである。

当初の設計では、YAML + Go `text/template` で shell script（`ralph.sh`）を生成し、その shell script を実行する方式を採用していた。しかしレビューの過程で以下の問題が明らかになった。

1. shell script に汎用 runtime 関数（`if` 式評価、auto commit、completion check 等）を埋め込むと、プロジェクト固有の差分より汎用コードの方が大きくなる
2. 汎用関数を `ralph` サブコマンド（`ralph eval-if`, `ralph auto-commit` 等）に分離すると、shell script は薄いグルーコードに過ぎなくなり、生成物としての存在意義が失われる
3. generation-time（Go template 展開）と runtime（shell 実行）の責務境界が曖昧になり、`if` 式の評価主体や runtime helper の設計に矛盾が生じた

## Decision

shell script 生成を廃止し、Go runner が `.ralph/config.yml` を直接読み取りタスクループを実行する方式を採用する。

- CLI は `ralph init`（雛形初期化）と `ralph run`（ループ実行）の 2 コマンドで構成する
- `ralph generate` コマンドと `ralph.sh` テンプレートは提供しない
- `if` 式は Go 内部 evaluator で直接評価する（`ralph eval-if` サブコマンドは不要）
- JSON 処理（`prd.json` の before/after 比較等）は Go で型安全に実装する（`jq` 依存は不要）
- step の `run` コマンドは `os/exec` で `sh -c` として子プロセス実行する
- `ralph run --dry-run` で実行計画を事前確認できるようにする

## Consequences

### Positive
- template 管理が不要になり、保守対象が減る
- generation-time と runtime の境界問題が解消される
- `jq` への runtime 依存がなくなる
- `if` 式評価や JSON 処理を Go 型安全に実装できる
- `--dry-run` で実行計画を事前確認できる

### Negative
- 生成された shell script を直接読んで「何が実行されるか」を確認する手段がなくなる（`--dry-run` で緩和）
- runner ロジックの変更にはバイナリの再ビルドが必要になる

### Neutral
- config schema、step 仕様、`if` 式仕様、auto_commit ロジック、completion 判定、exit code 体系は変更なし
- `ralph init` の雛形生成は `go:embed` テンプレートで従来どおり機能する
