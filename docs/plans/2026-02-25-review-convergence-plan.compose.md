# Review Convergence Plan Compose Pack

## Compose Reconstruction

### Reconstructed Design Summary

- config は `agent.run_command` を中心に command 別 completion profile（`completion.run`/`completion.review`）へ移行される。
- CLI は `ralph run` と `ralph review` を同一 `.ralph/config.yml` で切り替える設計となり、review mode は `review_convergence` strategy に限定される。
- runner は mode/role-aware 実行に拡張され、prompt path は `prompt.run.md` / `prompt.review.md` / `prompt.judge.md` へ分離される。
- review mode は scheduler state（review/judge counters + `run_id`）と artifact 規約（`REVIEW_XXXX.md`, `JUDGE_XXXX.json`）で反復収斂を管理する。
- 収斂判定は judge JSON 契約（`signal`, `new_findings`, `new_finding_keys` 必須）に基づき、`stable_rounds` と `min_reviews` の AND 条件で成功終了する。
- 未収斂時は `max_reviews` 到達で既存 exit `23` を返し、`tail_match` run mode の既存停止判定は維持される。
- `ralph init` は prompt 3 分割 + run/review profile 入り config を生成し、`prompt.md` と `.ralph/reviews/` は生成しない。

### Scope Diff

- Missing from tasks: none
- Extra in tasks: none
- Ambiguous mappings: none

### Alignment Verdict

- PASS
- Required fixes: none
- Checked At: 2026-02-25
