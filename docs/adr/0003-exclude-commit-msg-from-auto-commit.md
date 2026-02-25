# 0003: Exclude `.ralph/.commit-msg` from Auto Commit

## Status

Accepted

## Context

`.ralph/.commit-msg` は auto_commit の commit message 入力として使う実行時制御ファイルであり、成果物ではない。

一方で auto_commit は `git add -A` 系の staging を行うため、運用や環境差分によって `.ralph/.commit-msg` 自体が commit に混入するリスクがある。これが発生すると、次 iteration への持ち越しや履歴ノイズの原因になる。

また、GPG 署名を必須化している環境では、署名エージェント（例: 1Password）側の再承認タイムアウトにより `git commit` が一時的に失敗することがある。これを常に hard fail とすると loop の自動進行が止まりやすい。

## Decision

auto_commit の staging/cleanup を以下に固定する。

- `together`:
  - commit message は `.ralph/.commit-msg`（非空時）または `git.fallback_message` で決定する。
  - staging は `git add -A` 後に `.ralph/.commit-msg` を index から外し、`.ralph/.commit-msg` を commit 対象から除外する。
- `split`:
  - commit message は先に `.ralph/.commit-msg`（非空時）または `git.fallback_message` で決定する。
  - 第1 commit は非`.ralph/` を対象に作成する（`git add -A` + `git restore --staged .ralph/`）。
  - 第2 commit は `.ralph/` を対象に作成する（`git add -A .ralph/` + `git restore --staged .ralph/.commit-msg`）。
  - 第2 commit の message は `chore(ralph): mark ${task_id} complete in PRD and progress` を使う。
- `git.fallback_no_gpg_sign`:
  - runtime default は `false` とする（未指定時は再試行しない）。
  - `ralph init` のテンプレートでは運用上の利便性を優先して `true` を出力する。
  - `true` の場合、`git commit` が GPG 署名エラーで失敗したときに限り、同一引数で `git commit --no-gpg-sign ...` を 1 回だけ再試行する。
  - `false` の場合、または GPG 署名以外の失敗理由の場合は再試行しない。
- auto_commit が成功または no-op で終了した場合、`.ralph/.commit-msg` を削除する（未存在は許容）。

## Consequences

### Positive

- `.ralph/.commit-msg` が commit 履歴へ混入しない。
- loop 間で古い commit message が持ち越されにくくなる。
- 一時的な GPG 署名承認切れで loop が停止しにくくなる（`fallback_no_gpg_sign=true` 時）。

### Negative

- auto_commit の git 操作手順が増え、テストケースも増える。
- `--no-gpg-sign` 再試行を許可した場合、環境によっては unsigned commit が作成される可能性がある。

### Neutral

- file-size/type などの追加ガードは引き続き v1 スコープ外で、他の生成物混入は `.gitignore` と運用で制御する。
- 署名ポリシーの最終判断はリポジトリ運用側で行う（テンプレート値や config で制御可能）。
