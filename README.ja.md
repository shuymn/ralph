# ralph

`ralph` は、`.ralph/config.yml` を SoT としてタスクループを実行する Go 製 CLI です。  
`ralph init` で雛形を作り、`ralph run` または `ralph review` でループを実行します。

[English README](README.md)

## 主な動作

- `.ralph/config.yml` を読み込んで runner を構成
- `.ralph/prd.json` の `branchName` を必須とし、`run` 開始時に対象ブランチへ `git switch`（未存在なら作成）
- `run` mode では `agent.run_command` を `sh -c` で実行し、`.ralph/prompt.run.md` を stdin で渡す
- `review` mode では `review`/`judge` role を自動スケジュールし、`.ralph/prompt.review.md` と `.ralph/prompt.judge.md` を使う
- `phases.pre` / `phases.post` は `run` mode のみ適用される（`review` mode では無視される）
- `post` phase（`run` modeのみ）で `uses: auto_commit` を使った自動コミットが可能
- 完了条件:
  - `run`: `completion.run.signal` を `completion.run.tail_lines` で tail-match 判定
  - `review`: `completion.review.review_convergence` と judge JSON 契約で収斂判定

## 前提

- Go (`go.mod`: `go 1.25.0`)
- `git`
- `sh` (POSIX shell)
- `agent.run_command` で指定するエージェント CLI（例: `claude`, `codex` など）

## インストール

```bash
go install github.com/shuymn/ralph@latest
```

インストール後は Go のバイナリ配置先（`$GOBIN` または `$(go env GOPATH)/bin`）に PATH が通っていることを確認してください。

## セットアップ

```bash
ralph init
```

`ralph init` は以下を作成します（既存ファイルは上書きせず skip）。

- `.ralph/config.yml`
- `.ralph/prompt.run.md`
- `.ralph/prompt.review.md`
- `.ralph/prompt.judge.md`
- `.ralph/prd.json`
- `.ralph/progress.md`

## 最短実行手順

1. `.ralph/prd.json` を更新する（`branchName` と story 内容を実プロジェクト向けに置換）
2. `.ralph/config.yml` の `agent.run_command` を実環境のコマンドに合わせる
3. dry-run で実行計画を確認する

```bash
ralph run --dry-run
```

4. 問題なければ実行する

```bash
ralph run
```

`prd.json` の最小例:

```json
{
  "project": "your-project",
  "plan": "docs/plans/2026-01-01-your-plan.md",
  "branchName": "feature/task-001",
  "stories": [
    {
      "id": "TASK-001",
      "passes": false,
      "deps": []
    }
  ]
}
```

## config.yml

`ralph init` が出力する最小構成:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/shuymn/ralph/main/schemas/config.schema.json
version: "1"

agent:
  run_command: "claude -p --dangerously-skip-permissions"

completion:
  run:
    strategy: tail_match
    signal: "<promise>COMPLETE</promise>"
    tail_lines: 20
  review:
    strategy: review_convergence
    signal: "<promise>COMPLETE</promise>"
    review_convergence:
      min_reviews: 3
      max_reviews: 10
      judge_every: 2
      stable_rounds: 2

git:
  commit: split
  fallback_message: "feat: implement task (auto-commit)"
  fallback_no_gpg_sign: true

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

デフォルト値（未指定時）:

- `agent.max_iterations`: `60`
- `agent.sleep_seconds`: `5`
- `completion.run.strategy`: `tail_match`
- `completion.run.signal`: `<promise>COMPLETE</promise>`
- `completion.run.tail_lines`: `20`
- `completion.review.strategy`: `review_convergence`
- `completion.review.signal`: `<promise>COMPLETE</promise>`
- `completion.review.review_convergence.min_reviews`: `3`
- `completion.review.review_convergence.max_reviews`: `10`
- `completion.review.review_convergence.judge_every`: `2`
- `completion.review.review_convergence.stable_rounds`: `2`
- `git.commit`: `split`
- `git.fallback_message`: `feat: implement task (auto-commit)`
- `git.fallback_no_gpg_sign`: `false`
- `step.if`: `success()`
- `step.on_fail`: `stop_loop`

step の制約:

- `name` は必須
- `run` または `uses` を必ず片方だけ指定
- `uses` は現状 `auto_commit` のみ対応
- `on_fail` は `continue` または `stop_loop`

`if` 式で使える関数:

- `always()`
- `success()`
- `failure()`
- `changed()`

演算子: `!`, `&&`, `||`, `(`, `)`（`true` / `false` も可）

## auto_commit

`git.commit` のモード:

- `split`: `.ralph/` とそれ以外を分離して最大 2 commit
- `together`: 全変更を 1 commit

コミットメッセージ解決:

- `.ralph/.commit-msg` が存在し、内容が空白のみでない場合はそれを使用
- それ以外は `git.fallback_message` を使用
- `git.fallback_no_gpg_sign` が `true` の場合、GPG 署名エラーで commit が失敗したときに
  `git commit --no-gpg-sign` で 1 回だけ再試行

## 終了コード

- `0`: 正常終了（完了）
- `1`: コマンド失敗（`init`/`run` の一般失敗）
- `2`: 使い方エラー
- `20`: `on_fail: stop_loop` により停止
- `21`: completion signal 不一致
- `22`: 検証/実行時エラー
- `23`: `max_iterations` 到達

## 開発コマンド

- `task build`: バイナリビルド
- `task run`: ビルドして実行
- `task schema`: `schemas/config.schema.json` を再生成
- `task schema:check`: schema のドリフト確認
- `task fmt`: フォーマット
- `task lint`: lint 実行
- `task test`: `go test -race -shuffle=on -count=10 ./...`
- `task check`: `schema:check -> lint -> build -> test`
