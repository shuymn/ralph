package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/shuymn/ralph/internal/config"
	"github.com/shuymn/ralph/internal/prd"
)

const (
	ralphDirectory               = ".ralph/"
	taskCompletionMessagePattern = "chore(ralph): mark %s complete in PRD and progress"
)

var (
	errUnknownCommitMode = errors.New("unknown commit mode")
	errGitRunnerRequired = errors.New("git runner is required")
)

type Runner interface {
	Run(ctx context.Context, workDir string, args ...string) (string, error)
}

type AutoCommitOptions struct {
	WorkDir           string
	Mode              string
	FallbackMessage   string
	CommitMessagePath string
	PRDPath           string
	BeforePRD         prd.Document
	Runner            Runner
}

func AutoCommit(ctx context.Context, opts AutoCommitOptions) error {
	resolved := resolveOptions(opts)
	runner := resolved.Runner
	if runner == nil {
		return errGitRunnerRequired
	}

	switch resolved.Mode {
	case config.DefaultCommitMode:
		return runSplitMode(ctx, runner, resolved)
	case config.CommitModeTogether:
		return runTogetherMode(ctx, runner, resolved)
	default:
		return fmt.Errorf("%w %q", errUnknownCommitMode, resolved.Mode)
	}
}

func resolveOptions(opts AutoCommitOptions) AutoCommitOptions {
	resolved := opts
	resolved.Mode = strings.TrimSpace(resolved.Mode)
	if resolved.Mode == "" {
		resolved.Mode = config.DefaultCommitMode
	}

	if strings.TrimSpace(resolved.FallbackMessage) == "" {
		resolved.FallbackMessage = config.DefaultFallbackMessage
	}

	if resolved.Runner == nil {
		resolved.Runner = execRunner{}
	}

	return resolved
}

func runSplitMode(
	ctx context.Context,
	runner Runner,
	opts AutoCommitOptions,
) error {
	if err := runGit(ctx, runner, opts.WorkDir, "add", "-A", ralphDirectory); err != nil {
		return fmt.Errorf("stage .ralph files: %w", err)
	}

	hasStaged, err := hasStagedChanges(ctx, runner, opts.WorkDir)
	if err != nil {
		return fmt.Errorf("check staged .ralph changes: %w", err)
	}

	if hasStaged {
		after, err := prd.Load(opts.PRDPath)
		if err != nil {
			return fmt.Errorf("load prd for task_id extraction: %w", err)
		}

		taskID, err := ExtractTaskID(opts.BeforePRD, after)
		if err != nil {
			return fmt.Errorf("extract task_id: %w", err)
		}

		message := fmt.Sprintf(taskCompletionMessagePattern, taskID)
		if err := runGit(ctx, runner, opts.WorkDir, "commit", "-m", message); err != nil {
			return fmt.Errorf("commit .ralph changes: %w", err)
		}
	}

	if err := runGit(ctx, runner, opts.WorkDir, "add", "-A"); err != nil {
		return fmt.Errorf("stage all files: %w", err)
	}
	if err := runGit(ctx, runner, opts.WorkDir, "restore", "--staged", ralphDirectory); err != nil {
		return fmt.Errorf("unstage .ralph files: %w", err)
	}

	hasStaged, err = hasStagedChanges(ctx, runner, opts.WorkDir)
	if err != nil {
		return fmt.Errorf("check staged non-.ralph changes: %w", err)
	}
	if !hasStaged {
		return nil
	}

	messageArgs, err := resolveCommitMessageArgs(opts.CommitMessagePath, opts.FallbackMessage)
	if err != nil {
		return fmt.Errorf("resolve commit message: %w", err)
	}
	if err := runGit(
		ctx,
		runner,
		opts.WorkDir,
		append([]string{"commit"}, messageArgs...)...,
	); err != nil {
		return fmt.Errorf("commit non-.ralph changes: %w", err)
	}

	return nil
}

func runTogetherMode(
	ctx context.Context,
	runner Runner,
	opts AutoCommitOptions,
) error {
	if err := runGit(ctx, runner, opts.WorkDir, "add", "-A"); err != nil {
		return fmt.Errorf("stage all files: %w", err)
	}

	hasStaged, err := hasStagedChanges(ctx, runner, opts.WorkDir)
	if err != nil {
		return fmt.Errorf("check staged changes: %w", err)
	}
	if !hasStaged {
		return nil
	}

	messageArgs, err := resolveCommitMessageArgs(opts.CommitMessagePath, opts.FallbackMessage)
	if err != nil {
		return fmt.Errorf("resolve commit message: %w", err)
	}
	if err := runGit(
		ctx,
		runner,
		opts.WorkDir,
		append([]string{"commit"}, messageArgs...)...,
	); err != nil {
		return fmt.Errorf("commit staged changes: %w", err)
	}

	return nil
}

func hasStagedChanges(ctx context.Context, runner Runner, workDir string) (bool, error) {
	output, err := runGitOutput(ctx, runner, workDir, "diff", "--cached", "--name-only")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

func resolveCommitMessageArgs(path, fallback string) ([]string, error) {
	content, err := os.ReadFile(path)
	switch {
	case err == nil:
		if strings.TrimSpace(string(content)) != "" {
			return []string{"-F", path}, nil
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return nil, fmt.Errorf("read commit message file: %w", err)
	}

	return []string{"-m", fallback}, nil
}

func runGit(
	ctx context.Context,
	runner Runner,
	workDir string,
	args ...string,
) error {
	_, err := runGitOutput(ctx, runner, workDir, args...)
	return err
}

func runGitOutput(
	ctx context.Context,
	runner Runner,
	workDir string,
	args ...string,
) (string, error) {
	output, err := runner.Run(ctx, workDir, args...)
	if err != nil {
		return "", fmt.Errorf("run git %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), nil
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, trimmed)
}
