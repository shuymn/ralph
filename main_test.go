package main

import (
	"os"
	"path/filepath"
	"testing"

	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphrunner "github.com/shuymn/ralph/internal/runner"
)

func TestRunDispatchesDryRunCommands(t *testing.T) {
	testCases := []struct {
		name  string
		args  []string
		setup func(t *testing.T, root string)
	}{
		{
			name: "run dry-run uses run-mode prompts",
			args: []string{"run", "--dry-run"},
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeRalphFile(
					t,
					root,
					"prd.json",
					`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`+"\n",
				)
				writeRalphFile(t, root, "prompt.run.md", "run prompt\n")
			},
		},
		{
			name: "review dry-run uses review-mode prompts",
			args: []string{"review", "--dry-run"},
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeRalphFile(
					t,
					root,
					"prd.json",
					`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`+"\n",
				)
				writeRalphFile(t, root, "prompt.review.md", "review prompt\n")
				writeRalphFile(t, root, "prompt.judge.md", "judge prompt\n")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeRalphFile(t, root, "config.yml", validConfigYAML)
			tc.setup(t, root)
			t.Chdir(root)

			exitCode := run(tc.args)
			if exitCode != exitOK {
				t.Fatalf("expected exit code %d, got %d", exitOK, exitCode)
			}
		})
	}
}

func TestRunPropagatesExitCodes(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		setup    func(t *testing.T, root string)
		wantCode int
	}{
		{
			name: "propagates config validation error",
			args: []string{"run"},
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeRalphFile(t, root, "config.yml", "version: [\n")
			},
			wantCode: ralphconfig.ExitCodeValidation,
		},
		{
			name: "propagates runtime error from dry-run",
			args: []string{"run", "--dry-run"},
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeRalphFile(t, root, "config.yml", validConfigYAML)
				writeRalphFile(t, root, "prompt.run.md", "run prompt\n")
			},
			wantCode: ralphrunner.ExitCodeRuntime,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			tc.setup(t, root)
			t.Chdir(root)

			exitCode := run(tc.args)
			if exitCode != tc.wantCode {
				t.Fatalf("expected exit code %d, got %d", tc.wantCode, exitCode)
			}
		})
	}
}

func TestRunInitBranch(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(t *testing.T, root string)
		wantCode int
	}{
		{
			name: "init success",
			setup: func(t *testing.T, _ string) {
				t.Helper()
			},
			wantCode: exitOK,
		},
		{
			name: "init failure",
			setup: func(t *testing.T, root string) {
				t.Helper()
				if err := os.WriteFile(
					filepath.Join(root, ".ralph"),
					[]byte("file blocks directory creation\n"),
					0o600,
				); err != nil {
					t.Fatalf("write blocking .ralph file: %v", err)
				}
			},
			wantCode: exitFailure,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			tc.setup(t, root)
			t.Chdir(root)

			exitCode := run([]string{"init"})
			if exitCode != tc.wantCode {
				t.Fatalf("expected exit code %d, got %d", tc.wantCode, exitCode)
			}
		})
	}
}

func writeRalphFile(t *testing.T, root, name, content string) {
	t.Helper()

	ralphDir := filepath.Join(root, ".ralph")
	if err := os.MkdirAll(ralphDir, 0o755); err != nil {
		t.Fatalf("mkdir .ralph: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

const validConfigYAML = `version: "1"
agent:
  run_command: "echo noop"
`
