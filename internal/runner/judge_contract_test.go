package ralphrunner_test

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphrunner "github.com/shuymn/ralph/internal/runner"
)

func TestJudgeContractValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		judgeOutput string
	}{
		{
			name:        "missing signal",
			judgeOutput: `{"new_findings":0,"new_finding_keys":[]}` + "\n",
		},
		{
			name:        "new_findings must be integer",
			judgeOutput: `{"signal":"READY","new_findings":"0","new_finding_keys":[]}` + "\n",
		},
		{
			name:        "missing new_finding_keys",
			judgeOutput: `{"signal":"READY","new_findings":0}` + "\n",
		},
		{
			name:        "new_finding_keys must be string array",
			judgeOutput: `{"signal":"READY","new_findings":0,"new_finding_keys":"F1"}` + "\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			code, stderr := runReviewWithJudgeOutput(t, tc.judgeOutput)
			if code != ralphrunner.ExitCodeRuntime {
				t.Fatalf("expected runtime exit code %d, got %d", ralphrunner.ExitCodeRuntime, code)
			}
			if !strings.Contains(stderr, "judge") {
				t.Fatalf("expected judge contract error in stderr, got %q", stderr)
			}
		})
	}
}

func TestJudgeContractRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	code, stderr := runReviewWithJudgeOutput(
		t,
		`{"signal":"READY","new_findings":0,"new_finding_keys":[],"unexpected":true}`+"\n",
	)
	if code != ralphrunner.ExitCodeRuntime {
		t.Fatalf("expected runtime exit code %d, got %d", ralphrunner.ExitCodeRuntime, code)
	}
	if !strings.Contains(stderr, "decode judge artifact json") {
		t.Fatalf("expected decode error in stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, "unknown field") {
		t.Fatalf("expected unknown field detail in stderr, got %q", stderr)
	}
}

func TestJudgeContractRejectsNegativeNewFindings(t *testing.T) {
	t.Parallel()

	code, stderr := runReviewWithJudgeOutput(
		t,
		`{"signal":"READY","new_findings":-1,"new_finding_keys":[]}`+"\n",
	)
	if code != ralphrunner.ExitCodeRuntime {
		t.Fatalf("expected runtime exit code %d, got %d", ralphrunner.ExitCodeRuntime, code)
	}
	if !strings.Contains(stderr, "new_findings must be >= 0") {
		t.Fatalf("expected negative new_findings error in stderr, got %q", stderr)
	}
}

func runReviewWithJudgeOutput(t *testing.T, judgeOutput string) (int, string) {
	t.Helper()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		"prompt-run\n",
	)
	writeFile(t, filepath.Join(root, ".ralph", "prompt.review.md"), "review output\n")
	writeFile(t, filepath.Join(root, ".ralph", "prompt.judge.md"), judgeOutput)

	cfg := testConfig("cat", nil, nil)
	cfg.Agent.MaxIterations = 3
	cfg.Completion.Review.Signal = "READY"
	cfg.Completion.Review.ReviewConvergence = ralphconfig.ReviewConvergenceMode{
		MinReviews:   1,
		MaxReviews:   2,
		JudgeEvery:   1,
		StableRounds: 1,
	}

	var stderr bytes.Buffer
	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     &stderr,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
		Mode:       ralphrunner.ModeReview,
	})
	return code, stderr.String()
}
