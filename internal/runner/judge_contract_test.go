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

			root, tmpDir := setupWorkspace(
				t,
				`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
				"prompt-run\n",
			)
			writeFile(t, filepath.Join(root, ".ralph", "prompt.review.md"), "review output\n")
			writeFile(t, filepath.Join(root, ".ralph", "prompt.judge.md"), tc.judgeOutput)

			cfg := testConfig("cat", nil, nil)
			cfg.Agent.Command = ""
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
			if code != ralphrunner.ExitCodeRuntime {
				t.Fatalf("expected runtime exit code %d, got %d", ralphrunner.ExitCodeRuntime, code)
			}
			if !strings.Contains(stderr.String(), "judge") {
				t.Fatalf("expected judge contract error in stderr, got %q", stderr.String())
			}
		})
	}
}
