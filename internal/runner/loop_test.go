package ralphrunner_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
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
		`{"stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
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

func TestRunLogsStopLoopAndReturns20(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
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

	root, tmpDir := setupWorkspace(t, `{"stories":[]}`, "prompt\n")
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
		`{"stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
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
		`{"stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
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

func TestRunCleansTmpfilesOnSuccessFailureAndSignal(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		root, tmpDir := setupWorkspace(
			t,
			`{"stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
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
			`{"stories":[{"id":"TASK-1","passes":true,"deps":[]}]}`,
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
			`{"stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
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
	ralphDir := filepath.Join(root, ".ralph")
	if err := os.MkdirAll(ralphDir, 0o755); err != nil {
		t.Fatalf("mkdir .ralph failed: %v", err)
	}

	writeFile(t, filepath.Join(ralphDir, "prd.json"), prdJSON+"\n")
	writeFile(t, filepath.Join(ralphDir, "prompt.md"), prompt)

	tmpDir := t.TempDir()
	return root, tmpDir
}

func testConfig(
	agentCommand string,
	preSteps []ralphconfig.Step,
	postSteps []ralphconfig.Step,
) ralphconfig.Config {
	return ralphconfig.Config{
		Version: ralphconfig.SupportedVersion,
		Agent: ralphconfig.Agent{
			Command:       agentCommand,
			MaxIterations: 1,
			SleepSeconds:  0,
		},
		Completion: ralphconfig.Completion{
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
