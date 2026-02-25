# 0003: Exclude `.ralph/.commit-msg` from Auto Commit

## Status

Accepted

## Context

`.ralph/.commit-msg` は auto_commit の commit message 入力として使う実行時制御ファイルであり、成果物ではない。

一方で auto_commit は `git add -A` 系の staging を行うため、運用や環境差分によって `.ralph/.commit-msg` 自体が commit に混入するリスクがある。これが発生すると、次 iteration への持ち越しや履歴ノイズの原因になる。

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
- auto_commit が成功または no-op で終了した場合、`.ralph/.commit-msg` を削除する（未存在は許容）。

## Consequences

### Positive

- `.ralph/.commit-msg` が commit 履歴へ混入しない。
- loop 間で古い commit message が持ち越されにくくなる。

### Negative

- auto_commit の git 操作手順が増え、テストケースも増える。

### Neutral

- file-size/type などの追加ガードは引き続き v1 スコープ外で、他の生成物混入は `.gitignore` と運用で制御する。
