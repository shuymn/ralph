package runner_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuymn/ralph/internal/runner"
)

func TestDryRunOutputsPlanAndSkipsCommandExecution(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	preMarkerPath := filepath.Join(workDir, "pre-marker.txt")
	postMarkerPath := filepath.Join(workDir, "post-marker.txt")
	mainMarkerPath := filepath.Join(workDir, "main-marker.txt")

	configBody := buildConfig(configSpec{
		AgentCommand: "printf 'main\\n' >> " + shellQuote(mainMarkerPath) +
			"; printf '" + completionSignal + "\\n'",
		MaxIterations:  7,
		CompletionSign: "<promise>DONE</promise>",
		TailLines:      9,
		PreSteps: []stepSpec{
			{
				Name:   "pre_fmt",
				Run:    "printf 'pre\\n' >> " + shellQuote(preMarkerPath),
				IfExpr: "always()",
				OnFail: "continue",
			},
		},
		PostSteps: []stepSpec{
			{
				Name:   "post_lint",
				Run:    "printf 'post\\n' >> " + shellQuote(postMarkerPath),
				IfExpr: "failure() || changed()",
				OnFail: "stop_loop",
			},
			{
				Name:   "auto_commit",
				Uses:   "auto_commit",
				IfExpr: "changed()",
				OnFail: "stop_loop",
			},
		},
	})
	writeWorkspace(t, workDir, configBody, `{"stories":[{"id":"task-1","deps":[],"passes":false}]}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runner.DryRun(runner.Options{WorkDir: workDir, Stdout: &stdout, Stderr: &stderr})
	if code != runner.ExitCodeSuccess {
		t.Fatalf(
			"DryRun() exit code = %d, want %d, stderr=%q",
			code,
			runner.ExitCodeSuccess,
			stderr.String(),
		)
	}

	planOutput := stdout.String()
	requiredFragments := []string{
		"[ralph] dry-run execution plan",
		"command: \"printf 'main\\\\n' >> ",
		"max_iterations: 7",
		"sleep_seconds: 5",
		"strategy: \"tail_match\"",
		"signal: \"<promise>DONE</promise>\"",
		"tail_lines: 9",
		"commit: \"split\"",
		"fallback_message: \"fallback\"",
		"- name: \"pre_fmt\"",
		"run: \"printf 'pre\\\\n' >> ",
		"if: \"always()\"",
		"on_fail: \"continue\"",
		"- name: \"post_lint\"",
		"run: \"printf 'post\\\\n' >> ",
		"if: \"failure() || changed()\"",
		"- name: \"auto_commit\"",
		"uses: \"auto_commit\"",
		"stories: 1",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(planOutput, fragment) {
			t.Fatalf("dry-run output missing fragment %q in:\n%s", fragment, planOutput)
		}
	}

	for _, markerPath := range []string{preMarkerPath, postMarkerPath, mainMarkerPath} {
		if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
			t.Fatalf("marker file exists unexpectedly after dry-run: %s (err=%v)", markerPath, err)
		}
	}
}

func TestDryRunValidationParityWithRun(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		configBody string
		prdBody    string
	}{
		{
			name: "invalid if expression",
			configBody: buildConfig(configSpec{
				AgentCommand:  "printf 'noop\\n'",
				MaxIterations: 1,
				PreSteps: []stepSpec{
					{
						Name:   "invalid_if",
						Run:    "printf 'noop\\n'",
						IfExpr: "steps.build.success",
						OnFail: "stop_loop",
					},
				},
			}),
			prdBody: `{"stories":[{"id":"task-1","deps":[],"passes":false}]}`,
		},
		{
			name: "invalid prd",
			configBody: buildConfig(configSpec{
				AgentCommand:  "printf 'noop\\n'",
				MaxIterations: 1,
			}),
			prdBody: `{"stories":[]}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			workDir := t.TempDir()
			writeWorkspace(t, workDir, tc.configBody, tc.prdBody)

			dryRunCode := runner.DryRun(runner.Options{WorkDir: workDir})
			if dryRunCode != runner.ExitCodeConfigError {
				t.Fatalf(
					"DryRun() exit code = %d, want %d",
					dryRunCode,
					runner.ExitCodeConfigError,
				)
			}

			runCode := runner.Run(runner.Options{WorkDir: workDir})
			if runCode != runner.ExitCodeConfigError {
				t.Fatalf(
					"Run() exit code = %d, want %d",
					runCode,
					runner.ExitCodeConfigError,
				)
			}
		})
	}
}
