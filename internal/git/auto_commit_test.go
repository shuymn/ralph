package ralphgit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	ralphgit "github.com/shuymn/ralph/internal/git"
	ralphprd "github.com/shuymn/ralph/internal/prd"
)

func TestAutoCommitSplitStagesRalphBeforeNonRalph(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	ralphDir := filepath.Join(workspace, ".ralph")
	if err := os.MkdirAll(ralphDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	prdPath := filepath.Join(ralphDir, "prd.json")
	writeFile(t, prdPath, `{
  "branchName":"main",
  "stories":[{"id":"TASK-1","passes":true,"deps":[]}]
}`)

	commitMsgPath := filepath.Join(ralphDir, ".commit-msg")
	writeFile(t, commitMsgPath, "feat: implement task\n\nbody\n")

	diffCached := []string{"diff", "--cached", "--quiet", "--exit-code"}

	runner := newQueuedRunner()
	runner.enqueue([]string{"add", "-A", ".ralph/"}, ralphgit.CommandResult{ExitCode: 0})
	runner.enqueue(diffCached, ralphgit.CommandResult{ExitCode: 1})
	runner.enqueue(
		[]string{"commit", "-m", "chore(ralph): mark TASK-1 complete in PRD and progress"},
		ralphgit.CommandResult{ExitCode: 0},
	)
	runner.enqueue([]string{"add", "-A"}, ralphgit.CommandResult{ExitCode: 0})
	runner.enqueue(
		[]string{"restore", "--staged", ".ralph/"},
		ralphgit.CommandResult{ExitCode: 0},
	)
	runner.enqueue(diffCached, ralphgit.CommandResult{ExitCode: 1})
	runner.enqueue(
		[]string{"commit", "-F", commitMsgPath},
		ralphgit.CommandResult{ExitCode: 0},
	)

	err := ralphgit.AutoCommit(context.Background(), ralphgit.Options{
		WorkingDir:        workspace,
		Mode:              "split",
		FallbackMessage:   "feat: fallback",
		PRDPath:           prdPath,
		CommitMessagePath: commitMsgPath,
		BeforePRD: mustPRDDocument(t,
			`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`),
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("auto commit failed: %v", err)
	}

	assertCalls(t, runner.calls, workspace, [][]string{
		{"add", "-A", ".ralph/"},
		{"diff", "--cached", "--quiet", "--exit-code"},
		{"commit", "-m", "chore(ralph): mark TASK-1 complete in PRD and progress"},
		{"add", "-A"},
		{"restore", "--staged", ".ralph/"},
		{"diff", "--cached", "--quiet", "--exit-code"},
		{"commit", "-F", commitMsgPath},
	})
	runner.assertNoPending(t)
}

func TestAutoCommitTogetherStagesAllFiles(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	ralphDir := filepath.Join(workspace, ".ralph")
	if err := os.MkdirAll(ralphDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	commitMsgPath := filepath.Join(ralphDir, ".commit-msg")
	writeFile(t, commitMsgPath, "feat: from file\n\nbody\n")

	diffCached := []string{"diff", "--cached", "--quiet", "--exit-code"}

	runner := newQueuedRunner()
	runner.enqueue([]string{"add", "-A"}, ralphgit.CommandResult{ExitCode: 0})
	runner.enqueue(diffCached, ralphgit.CommandResult{ExitCode: 1})
	runner.enqueue(
		[]string{"commit", "-F", commitMsgPath},
		ralphgit.CommandResult{ExitCode: 0},
	)

	err := ralphgit.AutoCommit(context.Background(), ralphgit.Options{
		WorkingDir:        workspace,
		Mode:              "together",
		FallbackMessage:   "feat: fallback",
		PRDPath:           filepath.Join(ralphDir, "missing-prd.json"),
		CommitMessagePath: commitMsgPath,
		Runner:            runner,
	})
	if err != nil {
		t.Fatalf("auto commit failed: %v", err)
	}

	assertCalls(t, runner.calls, workspace, [][]string{
		{"add", "-A"},
		{"diff", "--cached", "--quiet", "--exit-code"},
		{"commit", "-F", commitMsgPath},
	})
	runner.assertNoPending(t)
}

func TestAutoCommitSplitRequiresExactlyOneTaskIDTransition(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		beforePRD string
		afterPRD  string
	}{
		{
			name:      "no transition",
			beforePRD: `{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
			afterPRD:  `{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		},
		{
			name:      "multiple transitions",
			beforePRD: `{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]},{"id":"TASK-2","passes":false,"deps":[]}]}`,
			afterPRD:  `{"branchName":"main","stories":[{"id":"TASK-1","passes":true,"deps":[]},{"id":"TASK-2","passes":true,"deps":[]}]}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			workspace := t.TempDir()
			ralphDir := filepath.Join(workspace, ".ralph")
			if err := os.MkdirAll(ralphDir, 0o755); err != nil {
				t.Fatalf("mkdir failed: %v", err)
			}

			prdPath := filepath.Join(ralphDir, "prd.json")
			writeFile(t, prdPath, tc.afterPRD)

			runner := newQueuedRunner()
			runner.enqueue([]string{"add", "-A", ".ralph/"}, ralphgit.CommandResult{ExitCode: 0})
			runner.enqueue(
				[]string{"diff", "--cached", "--quiet", "--exit-code"},
				ralphgit.CommandResult{ExitCode: 1},
			)

			err := ralphgit.AutoCommit(context.Background(), ralphgit.Options{
				WorkingDir:        workspace,
				Mode:              "split",
				FallbackMessage:   "feat: fallback",
				PRDPath:           prdPath,
				CommitMessagePath: filepath.Join(ralphDir, ".commit-msg"),
				BeforePRD:         mustPRDDocument(t, tc.beforePRD),
				Runner:            runner,
			})
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !strings.Contains(
				err.Error(),
				"expected exactly one story to transition passes false->true",
			) {
				t.Fatalf("unexpected error: %v", err)
			}

			assertCalls(t, runner.calls, workspace, [][]string{
				{"add", "-A", ".ralph/"},
				{"diff", "--cached", "--quiet", "--exit-code"},
			})
			runner.assertNoPending(t)
		})
	}
}

func TestAutoCommitNoopWhenStagedDiffIsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("split mode", func(t *testing.T) {
		t.Parallel()

		workspace := t.TempDir()
		runner := newQueuedRunner()
		runner.enqueue([]string{"add", "-A", ".ralph/"}, ralphgit.CommandResult{ExitCode: 0})
		runner.enqueue(
			[]string{"diff", "--cached", "--quiet", "--exit-code"},
			ralphgit.CommandResult{ExitCode: 0},
		)
		runner.enqueue([]string{"add", "-A"}, ralphgit.CommandResult{ExitCode: 0})
		runner.enqueue(
			[]string{"restore", "--staged", ".ralph/"},
			ralphgit.CommandResult{ExitCode: 0},
		)
		runner.enqueue(
			[]string{"diff", "--cached", "--quiet", "--exit-code"},
			ralphgit.CommandResult{ExitCode: 0},
		)

		err := ralphgit.AutoCommit(context.Background(), ralphgit.Options{
			WorkingDir:        workspace,
			Mode:              "split",
			FallbackMessage:   "feat: fallback",
			PRDPath:           filepath.Join(workspace, ".ralph", "prd.json"),
			CommitMessagePath: filepath.Join(workspace, ".ralph", ".commit-msg"),
			BeforePRD: mustPRDDocument(t,
				`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`),
			Runner: runner,
		})
		if err != nil {
			t.Fatalf("auto commit failed: %v", err)
		}

		assertCalls(t, runner.calls, workspace, [][]string{
			{"add", "-A", ".ralph/"},
			{"diff", "--cached", "--quiet", "--exit-code"},
			{"add", "-A"},
			{"restore", "--staged", ".ralph/"},
			{"diff", "--cached", "--quiet", "--exit-code"},
		})
		runner.assertNoPending(t)
	})

	t.Run("together mode", func(t *testing.T) {
		t.Parallel()

		workspace := t.TempDir()
		runner := newQueuedRunner()
		runner.enqueue([]string{"add", "-A"}, ralphgit.CommandResult{ExitCode: 0})
		runner.enqueue(
			[]string{"diff", "--cached", "--quiet", "--exit-code"},
			ralphgit.CommandResult{ExitCode: 0},
		)

		err := ralphgit.AutoCommit(context.Background(), ralphgit.Options{
			WorkingDir:        workspace,
			Mode:              "together",
			FallbackMessage:   "feat: fallback",
			PRDPath:           filepath.Join(workspace, ".ralph", "prd.json"),
			CommitMessagePath: filepath.Join(workspace, ".ralph", ".commit-msg"),
			Runner:            runner,
		})
		if err != nil {
			t.Fatalf("auto commit failed: %v", err)
		}

		assertCalls(t, runner.calls, workspace, [][]string{
			{"add", "-A"},
			{"diff", "--cached", "--quiet", "--exit-code"},
		})
		runner.assertNoPending(t)
	})
}

func TestAutoCommitUsesFallbackWhenCommitMsgIsEmpty(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		writeMessage bool
		content      string
	}{
		{name: "missing file"},
		{name: "whitespace only", writeMessage: true, content: " \n\t"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			workspace := t.TempDir()
			ralphDir := filepath.Join(workspace, ".ralph")
			if err := os.MkdirAll(ralphDir, 0o755); err != nil {
				t.Fatalf("mkdir failed: %v", err)
			}
			commitMsgPath := filepath.Join(ralphDir, ".commit-msg")
			if tc.writeMessage {
				writeFile(t, commitMsgPath, tc.content)
			}

			fallback := "feat: fallback message"
			runner := newQueuedRunner()
			runner.enqueue([]string{"add", "-A"}, ralphgit.CommandResult{ExitCode: 0})
			runner.enqueue(
				[]string{"diff", "--cached", "--quiet", "--exit-code"},
				ralphgit.CommandResult{ExitCode: 1},
			)
			runner.enqueue(
				[]string{"commit", "-m", fallback},
				ralphgit.CommandResult{ExitCode: 0},
			)

			err := ralphgit.AutoCommit(context.Background(), ralphgit.Options{
				WorkingDir:        workspace,
				Mode:              "together",
				FallbackMessage:   fallback,
				PRDPath:           filepath.Join(ralphDir, "prd.json"),
				CommitMessagePath: commitMsgPath,
				Runner:            runner,
			})
			if err != nil {
				t.Fatalf("auto commit failed: %v", err)
			}

			assertCalls(t, runner.calls, workspace, [][]string{
				{"add", "-A"},
				{"diff", "--cached", "--quiet", "--exit-code"},
				{"commit", "-m", fallback},
			})
			runner.assertNoPending(t)
		})
	}
}

type queuedRunner struct {
	calls     []gitCall
	responses map[string][]queuedResponse
}

type queuedResponse struct {
	result ralphgit.CommandResult
}

type gitCall struct {
	workingDir string
	args       []string
}

func newQueuedRunner() *queuedRunner {
	return &queuedRunner{
		responses: make(map[string][]queuedResponse),
	}
}

func (r *queuedRunner) Run(
	_ context.Context,
	workingDir string,
	args ...string,
) (ralphgit.CommandResult, error) {
	key := commandKey(args)
	queue := r.responses[key]
	if len(queue) == 0 {
		return ralphgit.CommandResult{}, fmt.Errorf("unexpected git command: %v", args)
	}
	response := queue[0]
	r.responses[key] = queue[1:]

	copiedArgs := append([]string(nil), args...)
	r.calls = append(r.calls, gitCall{
		workingDir: workingDir,
		args:       copiedArgs,
	})

	return response.result, nil
}

func (r *queuedRunner) enqueue(args []string, result ralphgit.CommandResult) {
	key := commandKey(args)
	r.responses[key] = append(r.responses[key], queuedResponse{
		result: result,
	})
}

func (r *queuedRunner) assertNoPending(t *testing.T) {
	t.Helper()

	for key, queue := range r.responses {
		if len(queue) == 0 {
			continue
		}
		t.Fatalf("remaining queued responses for command %q: %d", key, len(queue))
	}
}

func assertCalls(t *testing.T, calls []gitCall, workingDir string, expected [][]string) {
	t.Helper()

	if len(calls) != len(expected) {
		t.Fatalf("expected %d git calls, got %d", len(expected), len(calls))
	}

	for idx := range calls {
		if calls[idx].workingDir != workingDir {
			t.Fatalf(
				"call[%d] unexpected working dir: got=%q want=%q",
				idx,
				calls[idx].workingDir,
				workingDir,
			)
		}
		if !reflect.DeepEqual(calls[idx].args, expected[idx]) {
			t.Fatalf(
				"call[%d] args mismatch:\n got:  %v\n want: %v",
				idx,
				calls[idx].args,
				expected[idx],
			)
		}
	}
}

func commandKey(args []string) string {
	return strings.Join(args, "\x00")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func mustPRDDocument(t *testing.T, content string) ralphprd.Document {
	t.Helper()

	doc, err := ralphprd.ValidateBytes([]byte(content))
	if err != nil {
		t.Fatalf("validate prd failed: %v", err)
	}
	return doc
}
