package runner_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shuymn/ralph/internal/runner"
)

const completionSignal = "<promise>COMPLETE</promise>"

type configSpec struct {
	AgentCommand   string
	MaxIterations  int
	CompletionSign string
	TailLines      int
	PreSteps       []stepSpec
	PostSteps      []stepSpec
}

type stepSpec struct {
	Name   string
	Run    string
	Uses   string
	IfExpr string
	OnFail string
}

func TestRunPhaseOrderAndCompletion(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	orderPath := filepath.Join(workDir, "order.log")

	configBody := buildConfig(configSpec{
		AgentCommand: "printf 'main\\n' >> " + shellQuote(orderPath) +
			"; printf '" + completionSignal + "\\n'",
		MaxIterations: 2,
		PreSteps: []stepSpec{
			{
				Name:   "pre_step",
				Run:    "printf 'pre\\n' >> " + shellQuote(orderPath),
				IfExpr: "always()",
				OnFail: "stop_loop",
			},
		},
		PostSteps: []stepSpec{
			{
				Name:   "post_step",
				Run:    "printf 'post\\n' >> " + shellQuote(orderPath),
				IfExpr: "always()",
				OnFail: "stop_loop",
			},
		},
	})

	writeWorkspace(t, workDir, configBody, `{"stories":[{"id":"task-1","deps":[],"passes":true}]}`)

	var stderr bytes.Buffer
	code := runner.Run(runner.Options{WorkDir: workDir, Stderr: &stderr})
	if code != runner.ExitCodeSuccess {
		t.Fatalf(
			"Run() exit code = %d, want %d, stderr=%q",
			code,
			runner.ExitCodeSuccess,
			stderr.String(),
		)
	}

	order := mustReadFile(t, orderPath)
	if order != "pre\nmain\npost\n" {
		t.Fatalf("phase order = %q, want %q", order, "pre\\nmain\\npost\\n")
	}

	assertNoTmpFiles(t, workDir)
}

func TestRunStopLoopLogsContextAndReturns20(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	markerPath := filepath.Join(workDir, "should-not-exist.log")

	configBody := buildConfig(configSpec{
		AgentCommand:  "printf 'main\\n' >> " + shellQuote(markerPath),
		MaxIterations: 3,
		PreSteps: []stepSpec{
			{
				Name:   "pre_fail",
				Run:    "false",
				IfExpr: "always()",
				OnFail: "stop_loop",
			},
		},
		PostSteps: []stepSpec{
			{
				Name:   "post_step",
				Run:    "printf 'post\\n' >> " + shellQuote(markerPath),
				IfExpr: "always()",
				OnFail: "stop_loop",
			},
		},
	})

	writeWorkspace(t, workDir, configBody, `{"stories":[{"id":"task-1","deps":[],"passes":false}]}`)

	var stderr bytes.Buffer
	code := runner.Run(runner.Options{WorkDir: workDir, Stderr: &stderr})
	if code != runner.ExitCodeStopLoop {
		t.Fatalf("Run() exit code = %d, want %d", code, runner.ExitCodeStopLoop)
	}

	wantLogPrefix := "[ralph] stop_loop phase=pre step=pre_fail reason="
	if !strings.Contains(stderr.String(), wantLogPrefix) {
		t.Fatalf("stderr = %q, want prefix %q", stderr.String(), wantLogPrefix)
	}

	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatalf("marker file exists unexpectedly: stat err=%v", err)
	}
}

func TestRunExitCodeMappings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		configBody string
		prdBody    string
		wantCode   int
	}{
		{
			name: "invalid config returns 22",
			configBody: `version: "2"
agent:
  command: "printf 'x'"
git:
  commit: split
phases:
  pre:
    steps: []
  post:
    steps: []
`,
			prdBody:  `{"stories":[{"id":"task-1","deps":[],"passes":true}]}`,
			wantCode: runner.ExitCodeConfigError,
		},
		{
			name: "invalid prd returns 22",
			configBody: buildConfig(configSpec{
				AgentCommand:  "printf '" + completionSignal + "\\n'",
				MaxIterations: 1,
			}),
			prdBody:  `{"stories":[]}`,
			wantCode: runner.ExitCodeConfigError,
		},
		{
			name: "protocol mismatch returns 21",
			configBody: buildConfig(configSpec{
				AgentCommand:  "printf 'not-complete\\n'",
				MaxIterations: 1,
			}),
			prdBody:  `{"stories":[{"id":"task-1","deps":[],"passes":true}]}`,
			wantCode: runner.ExitCodeProtocolMismatch,
		},
		{
			name: "max iterations returns 23",
			configBody: buildConfig(configSpec{
				AgentCommand:  "printf 'incomplete\\n'",
				MaxIterations: 1,
			}),
			prdBody:  `{"stories":[{"id":"task-1","deps":[],"passes":false}]}`,
			wantCode: runner.ExitCodeMaxIterations,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			workDir := t.TempDir()
			writeWorkspace(t, workDir, tc.configBody, tc.prdBody)

			code := runner.Run(runner.Options{WorkDir: workDir})
			if code != tc.wantCode {
				t.Fatalf("Run() exit code = %d, want %d", code, tc.wantCode)
			}
		})
	}
}

func TestRunCleansTmpfileOnCompletionStopLoopAndSignal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		configBody string
		prdBody    string
		signalAt   time.Duration
		wantCode   int
	}{
		{
			name: "cleanup on completion",
			configBody: buildConfig(configSpec{
				AgentCommand:  "printf '" + completionSignal + "\\n'",
				MaxIterations: 2,
			}),
			prdBody:  `{"stories":[{"id":"task-1","deps":[],"passes":true}]}`,
			wantCode: runner.ExitCodeSuccess,
		},
		{
			name: "cleanup on stop_loop after main",
			configBody: buildConfig(configSpec{
				AgentCommand:  "printf '" + completionSignal + "\\n'",
				MaxIterations: 2,
				PostSteps: []stepSpec{
					{
						Name:   "post_fail",
						Run:    "false",
						IfExpr: "always()",
						OnFail: "stop_loop",
					},
				},
			}),
			prdBody:  `{"stories":[{"id":"task-1","deps":[],"passes":true}]}`,
			wantCode: runner.ExitCodeStopLoop,
		},
		{
			name: "cleanup on signal",
			configBody: buildConfig(configSpec{
				AgentCommand:  "sleep 5",
				MaxIterations: 1,
			}),
			prdBody:  `{"stories":[{"id":"task-1","deps":[],"passes":false}]}`,
			signalAt: 100 * time.Millisecond,
			wantCode: runner.ExitCodeStopLoop,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			workDir := t.TempDir()
			writeWorkspace(t, workDir, tc.configBody, tc.prdBody)

			var signals chan os.Signal
			if tc.signalAt > 0 {
				signals = make(chan os.Signal, 1)
				go func() {
					time.Sleep(tc.signalAt)
					signals <- os.Interrupt
				}()
			}

			code := runner.Run(runner.Options{WorkDir: workDir, Signals: signals})
			if code != tc.wantCode {
				t.Fatalf("Run() exit code = %d, want %d", code, tc.wantCode)
			}
			assertNoTmpFiles(t, workDir)
		})
	}
}

func buildConfig(spec configSpec) string {
	if spec.MaxIterations <= 0 {
		spec.MaxIterations = 1
	}
	if spec.CompletionSign == "" {
		spec.CompletionSign = completionSignal
	}
	if spec.TailLines <= 0 {
		spec.TailLines = 20
	}

	var builder strings.Builder
	builder.WriteString("version: \"1\"\n")
	builder.WriteString("agent:\n")
	fmt.Fprintf(&builder, "  command: %q\n", spec.AgentCommand)
	fmt.Fprintf(&builder, "  max_iterations: %d\n", spec.MaxIterations)
	builder.WriteString("  sleep_seconds: 0\n")
	builder.WriteString("completion:\n")
	builder.WriteString("  strategy: \"tail_match\"\n")
	fmt.Fprintf(&builder, "  signal: %q\n", spec.CompletionSign)
	fmt.Fprintf(&builder, "  tail_lines: %d\n", spec.TailLines)
	builder.WriteString("git:\n")
	builder.WriteString("  commit: \"split\"\n")
	builder.WriteString("  fallback_message: \"fallback\"\n")
	builder.WriteString("phases:\n")
	builder.WriteString("  pre:\n")
	writeSteps(&builder, spec.PreSteps)
	builder.WriteString("  post:\n")
	writeSteps(&builder, spec.PostSteps)

	return builder.String()
}

func writeSteps(builder *strings.Builder, steps []stepSpec) {
	if len(steps) == 0 {
		builder.WriteString("    steps: []\n")
		return
	}

	builder.WriteString("    steps:\n")
	for _, step := range steps {
		fmt.Fprintf(builder, "      - name: %q\n", step.Name)
		if step.Run != "" {
			fmt.Fprintf(builder, "        run: %q\n", step.Run)
		}
		if step.Uses != "" {
			fmt.Fprintf(builder, "        uses: %q\n", step.Uses)
		}
		if step.IfExpr != "" {
			fmt.Fprintf(builder, "        if: %q\n", step.IfExpr)
		}
		if step.OnFail != "" {
			fmt.Fprintf(builder, "        on_fail: %q\n", step.OnFail)
		}
	}
}

func writeWorkspace(t *testing.T, workDir, configBody, prdBody string) {
	t.Helper()

	ralphDir := filepath.Join(workDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	mustWriteFile(t, filepath.Join(ralphDir, "config.yml"), configBody)
	mustWriteFile(t, filepath.Join(ralphDir, "prompt.md"), "prompt for agent\n")
	mustWriteFile(t, filepath.Join(ralphDir, "prd.json"), prdBody)
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(content)
}

func shellQuote(input string) string {
	escaped := strings.ReplaceAll(input, "'", "'\\''")
	return "'" + escaped + "'"
}

func assertNoTmpFiles(t *testing.T, workDir string) {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(workDir, "ralph-main-*"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("tmp files remain: %v", files)
	}
}
