package git_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	ralphgit "github.com/shuymn/ralph/internal/git"
	"github.com/shuymn/ralph/internal/prd"
)

func TestAutoCommitSplitStagesRalphFirst(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	prdPath := filepath.Join(workDir, ".ralph", "prd.json")
	if err := os.MkdirAll(filepath.Dir(prdPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	before := prd.Document{
		Stories: []prd.Story{{ID: "task-1", Deps: []string{}, Passes: false}},
	}
	after := prd.Document{
		Stories: []prd.Story{{ID: "task-1", Deps: []string{}, Passes: true}},
	}
	mustWritePRD(t, prdPath, after)

	fallback := "feat: fallback"
	runner := &fakeRunner{
		t: t,
		expected: []expectedCommand{
			{args: []string{"add", "-A", ".ralph/"}},
			{args: []string{"diff", "--cached", "--name-only"}, stdout: ".ralph/prd.json\n"},
			{
				args: []string{
					"commit",
					"-m",
					"chore(ralph): mark task-1 complete in PRD and progress",
				},
			},
			{args: []string{"add", "-A"}},
			{args: []string{"restore", "--staged", ".ralph/"}},
			{
				args:   []string{"diff", "--cached", "--name-only"},
				stdout: "internal/runner/loop.go\n",
			},
			{args: []string{"commit", "-m", fallback}},
		},
	}

	err := ralphgit.AutoCommit(context.Background(), ralphgit.AutoCommitOptions{
		WorkDir:           workDir,
		Mode:              "split",
		FallbackMessage:   fallback,
		CommitMessagePath: filepath.Join(workDir, ".ralph", ".commit-msg"),
		PRDPath:           prdPath,
		BeforePRD:         before,
		Runner:            runner,
	})
	if err != nil {
		t.Fatalf("AutoCommit() error = %v", err)
	}

	runner.AssertDone()
}

func TestAutoCommitTogetherStagesAllFiles(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	fallback := "feat: fallback"
	runner := &fakeRunner{
		t: t,
		expected: []expectedCommand{
			{args: []string{"add", "-A"}},
			{args: []string{"diff", "--cached", "--name-only"}, stdout: "file.go\n"},
			{args: []string{"commit", "-m", fallback}},
		},
	}

	err := ralphgit.AutoCommit(context.Background(), ralphgit.AutoCommitOptions{
		WorkDir:           workDir,
		Mode:              "together",
		FallbackMessage:   fallback,
		CommitMessagePath: filepath.Join(workDir, ".ralph", ".commit-msg"),
		Runner:            runner,
	})
	if err != nil {
		t.Fatalf("AutoCommit() error = %v", err)
	}

	runner.AssertDone()
}

func TestExtractTaskIDRequiresExactlyOneFalseToTrue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		before  prd.Document
		after   prd.Document
		wantID  string
		wantErr bool
	}{
		{
			name: "single transition returns task id",
			before: prd.Document{
				Stories: []prd.Story{
					{ID: "task-1", Deps: []string{}, Passes: false},
					{ID: "task-2", Deps: []string{}, Passes: false},
				},
			},
			after: prd.Document{
				Stories: []prd.Story{
					{ID: "task-1", Deps: []string{}, Passes: true},
					{ID: "task-2", Deps: []string{}, Passes: false},
				},
			},
			wantID: "task-1",
		},
		{
			name: "no transition returns error",
			before: prd.Document{
				Stories: []prd.Story{{ID: "task-1", Deps: []string{}, Passes: false}},
			},
			after: prd.Document{
				Stories: []prd.Story{{ID: "task-1", Deps: []string{}, Passes: false}},
			},
			wantErr: true,
		},
		{
			name: "multiple transitions returns error",
			before: prd.Document{
				Stories: []prd.Story{
					{ID: "task-1", Deps: []string{}, Passes: false},
					{ID: "task-2", Deps: []string{}, Passes: false},
				},
			},
			after: prd.Document{
				Stories: []prd.Story{
					{ID: "task-1", Deps: []string{}, Passes: true},
					{ID: "task-2", Deps: []string{}, Passes: true},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotID, err := ralphgit.ExtractTaskID(tc.before, tc.after)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ExtractTaskID() error = nil, want non-nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ExtractTaskID() error = %v", err)
			}
			if gotID != tc.wantID {
				t.Fatalf("ExtractTaskID() = %q, want %q", gotID, tc.wantID)
			}
		})
	}
}

func TestAutoCommitSplitNoopWhenStagedDiffEmpty(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	runner := &fakeRunner{
		t: t,
		expected: []expectedCommand{
			{args: []string{"add", "-A", ".ralph/"}},
			{args: []string{"diff", "--cached", "--name-only"}},
			{args: []string{"add", "-A"}},
			{args: []string{"restore", "--staged", ".ralph/"}},
			{args: []string{"diff", "--cached", "--name-only"}},
		},
	}

	err := ralphgit.AutoCommit(context.Background(), ralphgit.AutoCommitOptions{
		WorkDir:           workDir,
		Mode:              "split",
		FallbackMessage:   "feat: fallback",
		CommitMessagePath: filepath.Join(workDir, ".ralph", ".commit-msg"),
		Runner:            runner,
		BeforePRD: prd.Document{
			Stories: []prd.Story{{ID: "task-1", Deps: []string{}, Passes: false}},
		},
		PRDPath: filepath.Join(workDir, ".ralph", "prd.json"),
	})
	if err != nil {
		t.Fatalf("AutoCommit() error = %v", err)
	}

	runner.AssertDone()
}

func TestAutoCommitUsesCommitMsgFileWhenPresent(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	commitMsgPath := filepath.Join(workDir, ".ralph", ".commit-msg")
	if err := os.MkdirAll(filepath.Dir(commitMsgPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	mustWriteFile(t, commitMsgPath, "feat: title\n\nbody line\n")

	runner := &fakeRunner{
		t: t,
		expected: []expectedCommand{
			{args: []string{"add", "-A"}},
			{args: []string{"diff", "--cached", "--name-only"}, stdout: "file.go\n"},
			{args: []string{"commit", "-F", commitMsgPath}},
		},
	}

	err := ralphgit.AutoCommit(context.Background(), ralphgit.AutoCommitOptions{
		WorkDir:           workDir,
		Mode:              "together",
		FallbackMessage:   "feat: fallback",
		CommitMessagePath: commitMsgPath,
		Runner:            runner,
	})
	if err != nil {
		t.Fatalf("AutoCommit() error = %v", err)
	}

	runner.AssertDone()
}

func TestAutoCommitFallsBackWhenCommitMsgTrimmedEmpty(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	commitMsgPath := filepath.Join(workDir, ".ralph", ".commit-msg")
	if err := os.MkdirAll(filepath.Dir(commitMsgPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	mustWriteFile(t, commitMsgPath, " \n\t\n")

	fallback := "feat: fallback"
	runner := &fakeRunner{
		t: t,
		expected: []expectedCommand{
			{args: []string{"add", "-A"}},
			{args: []string{"diff", "--cached", "--name-only"}, stdout: "file.go\n"},
			{args: []string{"commit", "-m", fallback}},
		},
	}

	err := ralphgit.AutoCommit(context.Background(), ralphgit.AutoCommitOptions{
		WorkDir:           workDir,
		Mode:              "together",
		FallbackMessage:   fallback,
		CommitMessagePath: commitMsgPath,
		Runner:            runner,
	})
	if err != nil {
		t.Fatalf("AutoCommit() error = %v", err)
	}

	runner.AssertDone()
}

type expectedCommand struct {
	args   []string
	stdout string
	err    error
}

type fakeRunner struct {
	t        *testing.T
	expected []expectedCommand
	calls    [][]string
}

func (f *fakeRunner) Run(_ context.Context, _ string, args ...string) (string, error) {
	f.t.Helper()

	call := append([]string(nil), args...)
	f.calls = append(f.calls, call)

	index := len(f.calls) - 1
	if index >= len(f.expected) {
		f.t.Fatalf("unexpected git command #%d: %v", index+1, args)
	}

	expected := f.expected[index]
	if !slices.Equal(args, expected.args) {
		f.t.Fatalf(
			"git command #%d = %v, want %v",
			index+1,
			args,
			expected.args,
		)
	}

	return expected.stdout, expected.err
}

func (f *fakeRunner) AssertDone() {
	f.t.Helper()

	if len(f.calls) != len(f.expected) {
		f.t.Fatalf(
			"git command count = %d, want %d",
			len(f.calls),
			len(f.expected),
		)
	}
}

func mustWritePRD(t *testing.T, path string, doc prd.Document) {
	t.Helper()

	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	mustWriteFile(t, path, string(body))
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func (f *fakeRunner) String() string {
	return fmt.Sprintf("calls=%v", f.calls)
}
