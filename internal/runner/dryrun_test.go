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

func TestDryRunPrintsExecutionPlanAndSkipsCommandExecution(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		"prompt\n",
	)

	agentMarker := filepath.Join(root, "agent-ran.marker")
	preMarker := filepath.Join(root, "pre-ran.marker")
	postMarker := filepath.Join(root, "post-ran.marker")

	cfg := ralphconfig.Config{
		Version: ralphconfig.SupportedVersion,
		Agent: ralphconfig.Agent{
			Command:       "printf 'agent-ran' > " + shQuote(agentMarker),
			MaxIterations: 7,
			SleepSeconds:  11,
		},
		Completion: ralphconfig.Completion{
			Strategy:  ralphconfig.DefaultCompletionStrategy,
			Signal:    "<done>COMPLETE</done>",
			TailLines: 9,
		},
		Git: ralphconfig.Git{
			Commit:          "together",
			FallbackMessage: "chore: fallback message",
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
		"agent.command:",
		"agent.max_iterations: 7",
		"agent.sleep_seconds: 11",
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
		"completion.strategy: tail_match",
		"completion.signal: <done>COMPLETE</done>",
		"completion.tail_lines: 9",
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

func TestDryRunReturns22WhenPRDValidationFails(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(t, `{"stories":[]}`, "prompt\n")
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
