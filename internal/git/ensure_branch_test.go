package ralphgit_test

import (
	"context"
	"strings"
	"testing"

	ralphgit "github.com/shuymn/ralph/internal/git"
)

func TestEnsureBranchNoopWhenAlreadyOnTargetBranch(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := newQueuedRunner()
	runner.enqueue(
		[]string{"branch", "--show-current"},
		ralphgit.CommandResult{ExitCode: 0, Stdout: "feature/task-1\n"},
	)

	err := ralphgit.EnsureBranch(context.Background(), ralphgit.EnsureBranchOptions{
		WorkingDir: workspace,
		BranchName: "feature/task-1",
		Runner:     runner,
	})
	if err != nil {
		t.Fatalf("ensure branch failed: %v", err)
	}

	assertCalls(t, runner.calls, workspace, [][]string{
		{"branch", "--show-current"},
	})
	runner.assertNoPending(t)
}

func TestEnsureBranchSwitchesToExistingBranch(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := newQueuedRunner()
	runner.enqueue(
		[]string{"branch", "--show-current"},
		ralphgit.CommandResult{ExitCode: 0, Stdout: "main\n"},
	)
	runner.enqueue(
		[]string{"show-ref", "--verify", "--quiet", "refs/heads/feature/task-1"},
		ralphgit.CommandResult{ExitCode: 0},
	)
	runner.enqueue(
		[]string{"switch", "feature/task-1"},
		ralphgit.CommandResult{ExitCode: 0},
	)

	err := ralphgit.EnsureBranch(context.Background(), ralphgit.EnsureBranchOptions{
		WorkingDir: workspace,
		BranchName: "feature/task-1",
		Runner:     runner,
	})
	if err != nil {
		t.Fatalf("ensure branch failed: %v", err)
	}

	assertCalls(t, runner.calls, workspace, [][]string{
		{"branch", "--show-current"},
		{"show-ref", "--verify", "--quiet", "refs/heads/feature/task-1"},
		{"switch", "feature/task-1"},
	})
	runner.assertNoPending(t)
}

func TestEnsureBranchCreatesMissingBranchFromMain(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := newQueuedRunner()
	runner.enqueue(
		[]string{"branch", "--show-current"},
		ralphgit.CommandResult{ExitCode: 0, Stdout: "main\n"},
	)
	runner.enqueue(
		[]string{"show-ref", "--verify", "--quiet", "refs/heads/feature/task-1"},
		ralphgit.CommandResult{ExitCode: 1},
	)
	runner.enqueue(
		[]string{"show-ref", "--verify", "--quiet", "refs/heads/main"},
		ralphgit.CommandResult{ExitCode: 0},
	)
	runner.enqueue(
		[]string{"switch", "-c", "feature/task-1", "main"},
		ralphgit.CommandResult{ExitCode: 0},
	)

	err := ralphgit.EnsureBranch(context.Background(), ralphgit.EnsureBranchOptions{
		WorkingDir: workspace,
		BranchName: "feature/task-1",
		Runner:     runner,
	})
	if err != nil {
		t.Fatalf("ensure branch failed: %v", err)
	}

	assertCalls(t, runner.calls, workspace, [][]string{
		{"branch", "--show-current"},
		{"show-ref", "--verify", "--quiet", "refs/heads/feature/task-1"},
		{"show-ref", "--verify", "--quiet", "refs/heads/main"},
		{"switch", "-c", "feature/task-1", "main"},
	})
	runner.assertNoPending(t)
}

func TestEnsureBranchCreatesMissingBranchFromCurrentWhenBaseMissing(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := newQueuedRunner()
	runner.enqueue(
		[]string{"branch", "--show-current"},
		ralphgit.CommandResult{ExitCode: 0, Stdout: "main\n"},
	)
	runner.enqueue(
		[]string{"show-ref", "--verify", "--quiet", "refs/heads/feature/task-1"},
		ralphgit.CommandResult{ExitCode: 1},
	)
	runner.enqueue(
		[]string{"show-ref", "--verify", "--quiet", "refs/heads/main"},
		ralphgit.CommandResult{ExitCode: 1},
	)
	runner.enqueue(
		[]string{"switch", "-c", "feature/task-1"},
		ralphgit.CommandResult{ExitCode: 0},
	)

	err := ralphgit.EnsureBranch(context.Background(), ralphgit.EnsureBranchOptions{
		WorkingDir: workspace,
		BranchName: "feature/task-1",
		Runner:     runner,
	})
	if err != nil {
		t.Fatalf("ensure branch failed: %v", err)
	}

	assertCalls(t, runner.calls, workspace, [][]string{
		{"branch", "--show-current"},
		{"show-ref", "--verify", "--quiet", "refs/heads/feature/task-1"},
		{"show-ref", "--verify", "--quiet", "refs/heads/main"},
		{"switch", "-c", "feature/task-1"},
	})
	runner.assertNoPending(t)
}

func TestEnsureBranchCreatesBaseBranchWithoutExplicitStartPoint(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := newQueuedRunner()
	runner.enqueue(
		[]string{"branch", "--show-current"},
		ralphgit.CommandResult{ExitCode: 0, Stdout: "master\n"},
	)
	runner.enqueue(
		[]string{"show-ref", "--verify", "--quiet", "refs/heads/main"},
		ralphgit.CommandResult{ExitCode: 1},
	)
	runner.enqueue(
		[]string{"switch", "-c", "main"},
		ralphgit.CommandResult{ExitCode: 0},
	)

	err := ralphgit.EnsureBranch(context.Background(), ralphgit.EnsureBranchOptions{
		WorkingDir: workspace,
		BranchName: "main",
		Runner:     runner,
	})
	if err != nil {
		t.Fatalf("ensure branch failed: %v", err)
	}

	assertCalls(t, runner.calls, workspace, [][]string{
		{"branch", "--show-current"},
		{"show-ref", "--verify", "--quiet", "refs/heads/main"},
		{"switch", "-c", "main"},
	})
	runner.assertNoPending(t)
}

func TestEnsureBranchReturnsErrorWhenSwitchFails(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := newQueuedRunner()
	runner.enqueue(
		[]string{"branch", "--show-current"},
		ralphgit.CommandResult{ExitCode: 0, Stdout: "main\n"},
	)
	runner.enqueue(
		[]string{"show-ref", "--verify", "--quiet", "refs/heads/feature/task-1"},
		ralphgit.CommandResult{ExitCode: 0},
	)
	runner.enqueue(
		[]string{"switch", "feature/task-1"},
		ralphgit.CommandResult{ExitCode: 128, Stderr: "conflict\n"},
	)

	err := ralphgit.EnsureBranch(context.Background(), ralphgit.EnsureBranchOptions{
		WorkingDir: workspace,
		BranchName: "feature/task-1",
		Runner:     runner,
	})
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "switch feature/task-1") {
		t.Fatalf("unexpected error: %v", err)
	}
}
