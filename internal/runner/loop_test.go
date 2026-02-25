package ralphrunner_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphrunner "github.com/shuymn/ralph/internal/runner"
)

func TestRunExecutesPreMainPostInOrder(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
		"prompt-line\n",
	)

	orderLog := filepath.Join(root, "order.log")
	mainInput := filepath.Join(root, ".ralph", "main-input.txt")

	cfg := testConfig(
		fmt.Sprintf(
			"printf 'main\\n' >> %s; cat > %s; printf '%s\\n'",
			shQuote(orderLog),
			shQuote(mainInput),
			ralphconfig.DefaultCompletionSignal,
		),
		[]ralphconfig.Step{{
			Name:   "pre_step",
			Run:    "printf 'pre\\n' >> " + shQuote(orderLog),
			If:     "always()",
			OnFail: "stop_loop",
		}},
		[]ralphconfig.Step{{
			Name:   "post_step",
			Run:    "printf 'post\\n' >> " + shQuote(orderLog),
			If:     "always()",
			OnFail: "stop_loop",
		}},
	)

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	order := readFile(t, orderLog)
	if order != "pre\nmain\npost\n" {
		t.Fatalf("unexpected phase order: %q", order)
	}

	stdin := readFile(t, mainInput)
	if stdin != "prompt-line\n" {
		t.Fatalf("expected prompt to be piped to main stdin, got %q", stdin)
	}
}

func TestRunUsesPromptRunPath(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
		"legacy-prompt\n",
	)

	runPrompt := filepath.Join(root, ".ralph", "prompt.run.md")
	writeFile(t, runPrompt, "run-prompt\n")

	mainInput := filepath.Join(root, ".ralph", "main-input.txt")
	cfg := testConfig(
		fmt.Sprintf(
			"cat > %s; printf '%s\\n'",
			shQuote(mainInput),
			ralphconfig.DefaultCompletionSignal,
		),
		nil,
		nil,
	)
	cfg.Agent.RunCommand = cfg.Agent.Command
	cfg.Agent.Command = ""

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
		Mode:       ralphrunner.ModeRun,
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	stdin := readFile(t, mainInput)
	if stdin != "run-prompt\n" {
		t.Fatalf("expected run prompt to be piped to main stdin, got %q", stdin)
	}
}

func TestReviewRequiresPromptFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                string
		presentPromptFile   string
		presentPromptValue  string
		missingPromptSubstr string
	}{
		{
			name:                "missing review prompt",
			presentPromptFile:   "prompt.judge.md",
			presentPromptValue:  "judge-prompt\n",
			missingPromptSubstr: "prompt.review.md",
		},
		{
			name:                "missing judge prompt",
			presentPromptFile:   "prompt.review.md",
			presentPromptValue:  "review-prompt\n",
			missingPromptSubstr: "prompt.judge.md",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root, tmpDir := setupWorkspace(
				t,
				`{"branchName":"main","stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
				"legacy-prompt\n",
			)
			writeFile(t, filepath.Join(root, ".ralph", tc.presentPromptFile), tc.presentPromptValue)

			cfg := testConfig("printf '"+ralphconfig.DefaultCompletionSignal+"\\n'", nil, nil)
			cfg.Agent.RunCommand = cfg.Agent.Command
			cfg.Agent.Command = ""
			cfg.Completion.Run = ralphconfig.RunCompletionProfile{
				Strategy:  ralphconfig.DefaultRunCompletionStrategy,
				Signal:    ralphconfig.DefaultCompletionSignal,
				TailLines: ralphconfig.DefaultRunCompletionTailLines,
			}
			cfg.Completion.Review = ralphconfig.ReviewCompletionProfile{
				Strategy: ralphconfig.DefaultReviewCompletionStrategy,
				Signal:   ralphconfig.DefaultCompletionSignal,
				ReviewConvergence: ralphconfig.ReviewConvergenceMode{
					MinReviews:   ralphconfig.DefaultReviewMinReviews,
					MaxReviews:   ralphconfig.DefaultReviewMaxReviews,
					JudgeEvery:   ralphconfig.DefaultReviewJudgeEvery,
					StableRounds: ralphconfig.DefaultReviewStableRounds,
				},
			}
			cfg.Completion.Strategy = ""
			cfg.Completion.Signal = ""
			cfg.Completion.TailLines = 0

			var stderr bytes.Buffer
			code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
				WorkingDir: root,
				Stdout:     io.Discard,
				Stderr:     &stderr,
				TempDir:    tmpDir,
				Sleep:      func(time.Duration) {},
				Mode:       ralphrunner.ModeReview,
			})
			if code != ralphrunner.ExitCodeRuntime {
				t.Fatalf("expected runtime exit code %d, got %d", ralphrunner.ExitCodeRuntime, code)
			}
			if !strings.Contains(stderr.String(), tc.missingPromptSubstr) {
				t.Fatalf(
					"expected missing prompt error for %s, got: %q",
					tc.missingPromptSubstr,
					stderr.String(),
				)
			}
		})
	}
}

func TestRoleCommandFallback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		role        ralphrunner.Role
		promptFile  string
		promptValue string
	}{
		{
			name:        "review role falls back to run_command",
			role:        ralphrunner.RoleReview,
			promptFile:  "prompt.review.md",
			promptValue: "review prompt\n",
		},
		{
			name:        "judge role falls back to run_command",
			role:        ralphrunner.RoleJudge,
			promptFile:  "prompt.judge.md",
			promptValue: "judge prompt\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root, tmpDir := setupWorkspace(
				t,
				`{"branchName":"main","stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
				"legacy-prompt\n",
			)

			writeFile(t, filepath.Join(root, ".ralph", "prompt.review.md"), "review prompt\n")
			writeFile(t, filepath.Join(root, ".ralph", "prompt.judge.md"), "judge prompt\n")
			writeFile(t, filepath.Join(root, ".ralph", "prompt.run.md"), "run prompt\n")

			stdinCapture := filepath.Join(root, ".ralph", "main-input.txt")
			cfg := testConfig("", nil, nil)
			cfg.Agent.RunCommand = fmt.Sprintf(
				"cat > %s; printf '%s\\n'",
				shQuote(stdinCapture),
				ralphconfig.DefaultCompletionSignal,
			)
			cfg.Agent.ReviewCommand = ""
			cfg.Agent.JudgeCommand = ""
			cfg.Agent.Command = ""
			cfg.Completion.Run = ralphconfig.RunCompletionProfile{
				Strategy:  ralphconfig.DefaultRunCompletionStrategy,
				Signal:    ralphconfig.DefaultCompletionSignal,
				TailLines: ralphconfig.DefaultRunCompletionTailLines,
			}
			cfg.Completion.Review = ralphconfig.ReviewCompletionProfile{
				Strategy: ralphconfig.DefaultReviewCompletionStrategy,
				Signal:   ralphconfig.DefaultCompletionSignal,
				ReviewConvergence: ralphconfig.ReviewConvergenceMode{
					MinReviews:   ralphconfig.DefaultReviewMinReviews,
					MaxReviews:   ralphconfig.DefaultReviewMaxReviews,
					JudgeEvery:   ralphconfig.DefaultReviewJudgeEvery,
					StableRounds: ralphconfig.DefaultReviewStableRounds,
				},
			}
			cfg.Completion.Strategy = ""
			cfg.Completion.Signal = ""
			cfg.Completion.TailLines = 0

			code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
				WorkingDir: root,
				Stdout:     io.Discard,
				Stderr:     io.Discard,
				TempDir:    tmpDir,
				Sleep:      func(time.Duration) {},
				Mode:       ralphrunner.ModeReview,
				Role:       tc.role,
			})
			if code != 0 {
				t.Fatalf("expected exit code 0, got %d", code)
			}

			got := readFile(t, stdinCapture)
			if got != tc.promptValue {
				t.Fatalf("expected prompt from %s, got %q", tc.promptFile, got)
			}
		})
	}
}

func TestRunTailMatchCompatibility(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
		"legacy-prompt\n",
	)
	writeFile(t, filepath.Join(root, ".ralph", "prompt.run.md"), "run prompt\n")

	cfg := ralphconfig.Config{
		Version: ralphconfig.SupportedVersion,
		Agent: ralphconfig.Agent{
			RunCommand: fmt.Sprintf(
				"printf 'noise\\n%s\\n'",
				ralphconfig.DefaultCompletionSignal,
			),
			MaxIterations: 1,
			SleepSeconds:  0,
		},
		Completion: ralphconfig.Completion{
			Run: ralphconfig.RunCompletionProfile{
				Strategy:  ralphconfig.DefaultRunCompletionStrategy,
				Signal:    ralphconfig.DefaultCompletionSignal,
				TailLines: 2,
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
			Commit:          ralphconfig.DefaultGitCommitMode,
			FallbackMessage: ralphconfig.DefaultFallbackCommit,
		},
	}

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
		Mode:       ralphrunner.ModeRun,
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunLogsStopLoopAndReturns20(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		"prompt\n",
	)

	cfg := testConfig("printf 'ignored\\n'", []ralphconfig.Step{{
		Name:   "pre_fail",
		Run:    "exit 1",
		If:     "always()",
		OnFail: "stop_loop",
	}}, nil)

	var stderr bytes.Buffer
	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     &stderr,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
	})
	if code != ralphrunner.ExitCodeStopLoop {
		t.Fatalf("expected stop-loop exit code %d, got %d", ralphrunner.ExitCodeStopLoop, code)
	}

	out := stderr.String()
	if !strings.Contains(out, "[ralph] stop_loop phase=pre step=pre_fail reason=") {
		t.Fatalf("missing stop_loop log: %q", out)
	}
}

func TestRunReturns22WhenPRDValidationFails(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(t, `{"branchName":"main","stories":[]}`, "prompt\n")
	cfg := testConfig("printf 'ignored\\n'", nil, nil)

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
	})
	if code != ralphrunner.ExitCodeRuntime {
		t.Fatalf("expected runtime exit code %d, got %d", ralphrunner.ExitCodeRuntime, code)
	}
}

func TestRunReturns21OnCompletionProtocolMismatch(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
		"prompt\n",
	)

	cfg := testConfig("printf 'done without signal\\n'", nil, nil)

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
	})
	if code != ralphrunner.ExitCodeCompletionMismatch {
		t.Fatalf(
			"expected completion mismatch exit code %d, got %d",
			ralphrunner.ExitCodeCompletionMismatch,
			code,
		)
	}
}

func TestRunReturns23WhenMaxIterationsReached(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		"prompt\n",
	)

	cfg := testConfig("printf 'no completion\\n'", nil, nil)

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
	})
	if code != ralphrunner.ExitCodeMaxIterations {
		t.Fatalf(
			"expected max-iterations exit code %d, got %d",
			ralphrunner.ExitCodeMaxIterations,
			code,
		)
	}
}

func TestRunSwitchesToBranchFromPRD(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"feature/task-1","stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
		"prompt\n",
	)

	cfg := testConfig("printf '"+ralphconfig.DefaultCompletionSignal+"\\n'", nil, nil)

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	currentBranch, err := gitOutput(root, "branch", "--show-current")
	if err != nil {
		t.Fatalf("read current branch failed: %v", err)
	}
	if currentBranch != "feature/task-1" {
		t.Fatalf("unexpected branch after run: got=%q want=%q", currentBranch, "feature/task-1")
	}
}

func TestRunCleansTmpfilesOnSuccessFailureAndSignal(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		root, tmpDir := setupWorkspace(
			t,
			`{"branchName":"main","stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
			"prompt\n",
		)
		cfg := testConfig(
			"printf '"+ralphconfig.DefaultCompletionSignal+"\\n'",
			nil,
			nil,
		)

		code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
			WorkingDir: root,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			TempDir:    tmpDir,
			Sleep:      func(time.Duration) {},
		})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
		assertNoTmpfiles(t, tmpDir)
	})

	t.Run("failure", func(t *testing.T) {
		t.Parallel()

		root, tmpDir := setupWorkspace(
			t,
			`{"branchName":"main","stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
			"prompt\n",
		)
		cfg := testConfig("printf 'missing signal\\n'", nil, nil)

		code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
			WorkingDir: root,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			TempDir:    tmpDir,
			Sleep:      func(time.Duration) {},
		})
		if code != ralphrunner.ExitCodeCompletionMismatch {
			t.Fatalf(
				"expected completion mismatch exit code %d, got %d",
				ralphrunner.ExitCodeCompletionMismatch,
				code,
			)
		}
		assertNoTmpfiles(t, tmpDir)
	})

	t.Run("signal", func(t *testing.T) {
		t.Parallel()

		root, tmpDir := setupWorkspace(
			t,
			`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
			"prompt\n",
		)
		cfg := testConfig("sleep 5", nil, nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		code := ralphrunner.Run(ctx, cfg, ralphrunner.Options{
			WorkingDir: root,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			TempDir:    tmpDir,
			Sleep:      func(time.Duration) {},
		})
		if code != ralphrunner.ExitCodeRuntime {
			t.Fatalf("expected runtime exit code %d, got %d", ralphrunner.ExitCodeRuntime, code)
		}
		assertNoTmpfiles(t, tmpDir)
	})
}

func setupWorkspace(t *testing.T, prdJSON, prompt string) (string, string) {
	t.Helper()

	root := t.TempDir()
	initGitRepo(t, root)

	ralphDir := filepath.Join(root, ".ralph")
	if err := os.MkdirAll(ralphDir, 0o755); err != nil {
		t.Fatalf("mkdir .ralph failed: %v", err)
	}

	writeFile(t, filepath.Join(ralphDir, "prd.json"), prdJSON+"\n")
	writeFile(t, filepath.Join(ralphDir, "prompt.run.md"), prompt)

	tmpDir := t.TempDir()
	return root, tmpDir
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()

	if err := runGit(root, "init", "--initial-branch=main"); err != nil {
		if err := runGit(root, "init"); err != nil {
			t.Fatalf("git init failed: %v", err)
		}
		if err := runGit(root, "switch", "-c", "main"); err != nil {
			t.Fatalf("git switch -c main failed: %v", err)
		}
	}
}

func runGit(root string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	return fmt.Errorf(
		"git %s: %w: %s",
		strings.Join(args, " "),
		err,
		strings.TrimSpace(string(output)),
	)
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"git %s: %w: %s",
			strings.Join(args, " "),
			err,
			strings.TrimSpace(string(output)),
		)
	}
	return strings.TrimSpace(string(output)), nil
}

func testConfig(
	agentCommand string,
	preSteps []ralphconfig.Step,
	postSteps []ralphconfig.Step,
) ralphconfig.Config {
	return ralphconfig.Config{
		Version: ralphconfig.SupportedVersion,
		Agent: ralphconfig.Agent{
			RunCommand:    agentCommand,
			Command:       agentCommand,
			MaxIterations: 1,
			SleepSeconds:  0,
		},
		Completion: ralphconfig.Completion{
			Run: ralphconfig.RunCompletionProfile{
				Strategy:  ralphconfig.DefaultRunCompletionStrategy,
				Signal:    ralphconfig.DefaultCompletionSignal,
				TailLines: ralphconfig.DefaultRunCompletionTailLines,
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
			Strategy:  ralphconfig.DefaultCompletionStrategy,
			Signal:    ralphconfig.DefaultCompletionSignal,
			TailLines: ralphconfig.DefaultCompletionTailLines,
		},
		Git: ralphconfig.Git{
			Commit:          ralphconfig.DefaultGitCommitMode,
			FallbackMessage: ralphconfig.DefaultFallbackCommit,
		},
		Phases: ralphconfig.Phases{
			Pre:  ralphconfig.Phase{Steps: preSteps},
			Post: ralphconfig.Phase{Steps: postSteps},
		},
	}
}

func assertNoTmpfiles(t *testing.T, tmpDir string) {
	t.Helper()

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("read temp dir failed: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("expected no tmpfiles, got: %v", names)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}
	return string(content)
}

func shQuote(value string) string {
	replaced := strings.ReplaceAll(value, "'", `'"'"'`)
	return "'" + replaced + "'"
}
