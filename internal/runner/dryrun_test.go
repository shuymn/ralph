package ralphrunner_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphrunner "github.com/shuymn/ralph/internal/runner"
)

func TestDryRunRunProfileOutput(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		"prompt\n",
	)

	agentMarker := filepath.Join(root, "agent-ran.marker")
	preMarker := filepath.Join(root, "pre-ran.marker")
	postMarker := filepath.Join(root, "post-ran.marker")

	cfg := ralphconfig.Config{
		Version: ralphconfig.SupportedVersion,
		Agent: ralphconfig.Agent{
			RunCommand:    "printf 'agent-ran' > " + shQuote(agentMarker),
			MaxIterations: 7,
			SleepSeconds:  11,
		},
		Completion: ralphconfig.Completion{
			Run: ralphconfig.RunCompletionProfile{
				Strategy:  ralphconfig.DefaultRunCompletionStrategy,
				Signal:    "<done>COMPLETE</done>",
				TailLines: 9,
			},
			Review: ralphconfig.ReviewCompletionProfile{
				Strategy: ralphconfig.DefaultReviewCompletionStrategy,
				Signal:   ralphconfig.DefaultCompletionSignal,
				ReviewConvergence: ralphconfig.ReviewConvergenceMode{
					MinReviews:   ralphconfig.DefaultReviewMinReviews,
					MaxReviews:   ralphconfig.DefaultReviewMaxReviews,
					JudgeEvery:   ralphconfig.DefaultReviewJudgeEvery,
					StableRounds: ralphconfig.DefaultReviewStableRounds,
				},
			},
		},
		Git: ralphconfig.Git{
			Commit:            "together",
			FallbackMessage:   "chore: fallback message",
			FallbackNoGPGSign: true,
		},
		Phases: ralphconfig.Phases{
			Pre: ralphconfig.Phase{
				Steps: []ralphconfig.Step{{
					Name:   "pre_check",
					Run:    "printf 'pre-ran' > " + shQuote(preMarker),
					If:     "always()",
					OnFail: "continue",
				}},
			},
			Post: ralphconfig.Phase{
				Steps: []ralphconfig.Step{
					{
						Name:   "tests",
						Run:    "printf 'post-ran' > " + shQuote(postMarker),
						If:     "success()",
						OnFail: "stop_loop",
					},
					{
						Name:   "auto_commit",
						Uses:   "auto_commit",
						If:     "changed()",
						OnFail: "stop_loop",
					},
				},
			},
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := ralphrunner.DryRun(cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     &stdout,
		Stderr:     &stderr,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	out := stdout.String()
	expectedSnippets := []string{
		"[ralph] dry-run execution plan",
		"mode: run",
		"agent.run_command:",
		"agent.max_iterations: 7",
		"agent.sleep_seconds: 11",
		"prompt.run_path:",
		"phases.pre:",
		"name: pre_check",
		"run: printf 'pre-ran'",
		"if: always()",
		"on_fail: continue",
		"phases.post:",
		"name: tests",
		"run: printf 'post-ran'",
		"name: auto_commit",
		"uses: auto_commit",
		"if: changed()",
		"on_fail: stop_loop",
		"git.commit: together",
		"git.fallback_message: chore: fallback message",
		"git.fallback_no_gpg_sign: true",
		"completion.run.strategy: tail_match",
		"completion.run.signal: <done>COMPLETE</done>",
		"completion.run.tail_lines: 9",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(out, snippet) {
			t.Fatalf("dry-run output missing %q\nfull output:\n%s", snippet, out)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	assertFileNotExists(t, agentMarker)
	assertFileNotExists(t, preMarker)
	assertFileNotExists(t, postMarker)
}

func TestDryRunReviewProfileOutput(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		"run-prompt\n",
	)
	writeFile(t, filepath.Join(root, ".ralph", "prompt.review.md"), "review-prompt\n")
	writeFile(t, filepath.Join(root, ".ralph", "prompt.judge.md"), "judge-prompt\n")

	runCommand := "printf 'run-agent\\n'"
	judgeCommand := "printf 'judge-agent\\n'"
	cfg := testConfig(runCommand, nil, nil)
	cfg.Agent.ReviewCommand = ""
	cfg.Agent.JudgeCommand = judgeCommand
	cfg.Completion.Review.Signal = "<done>REVIEW</done>"
	cfg.Completion.Review.ReviewConvergence = ralphconfig.ReviewConvergenceMode{
		MinReviews:   4,
		MaxReviews:   12,
		JudgeEvery:   3,
		StableRounds: 2,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := ralphrunner.DryRun(cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     &stdout,
		Stderr:     &stderr,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
		Mode:       ralphrunner.ModeReview,
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	out := stdout.String()
	expectedSnippets := []string{
		"[ralph] dry-run execution plan",
		"mode: review",
		"agent.run_command: " + runCommand,
		"agent.review_command.resolved: " + runCommand,
		"agent.review_command.source: fallback(run_command)",
		"agent.judge_command.resolved: " + judgeCommand,
		"agent.judge_command.source: judge_command",
		"prompt.review_path: " + filepath.Join(root, ".ralph", "prompt.review.md"),
		"prompt.judge_path: " + filepath.Join(root, ".ralph", "prompt.judge.md"),
		"phases: disabled in review mode",
		"completion.review.strategy: review_convergence",
		"completion.review.signal: <done>REVIEW</done>",
		"completion.review.review_convergence.min_reviews: 4",
		"completion.review.review_convergence.max_reviews: 12",
		"completion.review.review_convergence.judge_every: 3",
		"completion.review.review_convergence.stable_rounds: 2",
		"judge_json_contract.required_keys: signal,new_findings,new_finding_keys",
		"judge_json_contract.stable_condition: signal == completion.review.signal && new_findings == 0",
		"judge_stdin.format: MACHINE_CONTEXT_JSON_START/<json>/MACHINE_CONTEXT_JSON_END + prompt.judge.md",
		"judge_stdin.context_keys: run_id,review_count,judge_count,reviews_since_judge,completion_signal,new_review_files,previously_judged_review_files,all_review_files,current_judge_artifact,previous_judge_artifacts",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(out, snippet) {
			t.Fatalf("dry-run output missing %q\nfull output:\n%s", snippet, out)
		}
	}
	for _, forbidden := range []string{"phases.pre:", "phases.post:", "git.commit:"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf(
				"dry-run output must not include %q in review mode\nfull output:\n%s",
				forbidden,
				out,
			)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestDryRunReturns22WhenPRDValidationFails(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(t, `{"branchName":"main","stories":[]}`, "prompt\n")
	cfg := testConfig("printf 'main\\n'", nil, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := ralphrunner.DryRun(cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     &stdout,
		Stderr:     &stderr,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
	})
	if code != ralphrunner.ExitCodeRuntime {
		t.Fatalf("expected runtime exit code %d, got %d", ralphrunner.ExitCodeRuntime, code)
	}
	if !strings.Contains(stderr.String(), "validate prd") {
		t.Fatalf("expected PRD validation error in stderr, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func assertFileNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("expected file to not exist: %s", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("stat %s failed: %v", path, err)
	}
}
