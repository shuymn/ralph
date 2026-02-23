package ralphgit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	ralphprd "github.com/shuymn/ralph/internal/prd"
)

const (
	CommitModeSplit    = "split"
	CommitModeTogether = "together"

	ralphOnlyPath        = ".ralph/"
	ralphCommitMessage   = "chore(ralph): mark %s complete in PRD and progress"
	noStagedChangesCode  = 0
	hasStagedChangesCode = 1
)

var (
	errUnsupportedCommitMode = errors.New("unsupported git commit mode")
	errGitDiffStagedFailed   = errors.New("git diff --cached --quiet --exit-code failed")
	errGitCommandFailed      = errors.New("git command failed")
	errReadCommitMessageFile = errors.New("read commit message file")
	errReadPRDForTaskID      = errors.New("read prd for task_id extraction")
	errValidatePRDForTaskID  = errors.New("validate prd for task_id extraction")
)

type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type CommandRunner interface {
	Run(ctx context.Context, workingDir string, args ...string) (CommandResult, error)
}

type Options struct {
	WorkingDir        string
	Mode              string
	FallbackMessage   string
	PRDPath           string
	CommitMessagePath string
	BeforePRD         ralphprd.Document
	Runner            CommandRunner
}

func AutoCommit(ctx context.Context, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	opts = normalizeOptions(opts)

	switch opts.Mode {
	case CommitModeSplit:
		return autoCommitSplit(ctx, opts)
	case CommitModeTogether:
		return autoCommitTogether(ctx, opts)
	default:
		return fmt.Errorf("%w: mode=%q", errUnsupportedCommitMode, opts.Mode)
	}
}

func autoCommitSplit(ctx context.Context, opts Options) error {
	if err := runGitExpectSuccess(ctx, opts, "add", "-A", ralphOnlyPath); err != nil {
		return err
	}

	ralphChanged, err := hasStagedDiff(ctx, opts)
	if err != nil {
		return err
	}
	if ralphChanged {
		afterPRD, loadErr := loadPRD(opts.PRDPath)
		if loadErr != nil {
			return loadErr
		}

		taskID, taskErr := ExtractTaskID(opts.BeforePRD, afterPRD)
		if taskErr != nil {
			return taskErr
		}

		message := fmt.Sprintf(ralphCommitMessage, taskID)
		if err := runGitExpectSuccess(ctx, opts, "commit", "-m", message); err != nil {
			return err
		}
	}

	if err := runGitExpectSuccess(ctx, opts, "add", "-A"); err != nil {
		return err
	}
	if err := runGitExpectSuccess(ctx, opts, "restore", "--staged", ralphOnlyPath); err != nil {
		return err
	}

	otherChanged, err := hasStagedDiff(ctx, opts)
	if err != nil {
		return err
	}
	if !otherChanged {
		return nil
	}

	return commitNonRalphChanges(ctx, opts)
}

func autoCommitTogether(ctx context.Context, opts Options) error {
	if err := runGitExpectSuccess(ctx, opts, "add", "-A"); err != nil {
		return err
	}

	stagedChanges, err := hasStagedDiff(ctx, opts)
	if err != nil {
		return err
	}
	if !stagedChanges {
		return nil
	}

	return commitNonRalphChanges(ctx, opts)
}

func commitNonRalphChanges(ctx context.Context, opts Options) error {
	source, err := resolveCommitMessageSource(opts.CommitMessagePath, opts.FallbackMessage)
	if err != nil {
		return err
	}

	if source.useFile {
		return runGitExpectSuccess(ctx, opts, "commit", "-F", source.value)
	}
	return runGitExpectSuccess(ctx, opts, "commit", "-m", source.value)
}

func hasStagedDiff(ctx context.Context, opts Options) (bool, error) {
	result, err := opts.Runner.Run(
		ctx,
		opts.WorkingDir,
		"diff",
		"--cached",
		"--quiet",
		"--exit-code",
	)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errGitDiffStagedFailed, err)
	}

	switch result.ExitCode {
	case noStagedChangesCode:
		return false, nil
	case hasStagedChangesCode:
		return true, nil
	default:
		reason := strings.TrimSpace(result.Stderr)
		if reason == "" {
			reason = strings.TrimSpace(result.Stdout)
		}
		if reason == "" {
			return false, fmt.Errorf(
				"%w: exit_code=%d",
				errGitDiffStagedFailed,
				result.ExitCode,
			)
		}
		return false, fmt.Errorf(
			"%w: exit_code=%d reason=%s",
			errGitDiffStagedFailed,
			result.ExitCode,
			reason,
		)
	}
}

func runGitExpectSuccess(ctx context.Context, opts Options, args ...string) error {
	result, err := opts.Runner.Run(ctx, opts.WorkingDir, args...)
	if err != nil {
		return fmt.Errorf(
			"%w: command=%s: %w",
			errGitCommandFailed,
			strings.Join(args, " "),
			err,
		)
	}
	if result.ExitCode == 0 {
		return nil
	}

	reason := strings.TrimSpace(result.Stderr)
	if reason == "" {
		reason = strings.TrimSpace(result.Stdout)
	}
	if reason == "" {
		return fmt.Errorf(
			"%w: command=%s exit_code=%d",
			errGitCommandFailed,
			strings.Join(args, " "),
			result.ExitCode,
		)
	}
	return fmt.Errorf(
		"%w: command=%s exit_code=%d reason=%s",
		errGitCommandFailed,
		strings.Join(args, " "),
		result.ExitCode,
		reason,
	)
}

type commitMessageSource struct {
	useFile bool
	value   string
}

func resolveCommitMessageSource(path, fallback string) (commitMessageSource, error) {
	content, err := os.ReadFile(path)
	if err == nil {
		if strings.TrimSpace(string(content)) != "" {
			return commitMessageSource{
				useFile: true,
				value:   path,
			}, nil
		}
		return commitMessageSource{
			useFile: false,
			value:   fallback,
		}, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return commitMessageSource{
			useFile: false,
			value:   fallback,
		}, nil
	}
	return commitMessageSource{}, fmt.Errorf("%w: %w", errReadCommitMessageFile, err)
}

func loadPRD(path string) (ralphprd.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ralphprd.Document{}, fmt.Errorf("%w: %w", errReadPRDForTaskID, err)
	}

	doc, err := ralphprd.ValidateBytes(content)
	if err != nil {
		return ralphprd.Document{}, fmt.Errorf("%w: %w", errValidatePRDForTaskID, err)
	}
	return doc, nil
}

func normalizeOptions(opts Options) Options {
	if strings.TrimSpace(opts.Mode) == "" {
		opts.Mode = CommitModeSplit
	}
	if opts.Runner == nil {
		opts.Runner = cliRunner{}
	}
	return opts
}

type cliRunner struct{}

func (cliRunner) Run(
	ctx context.Context,
	workingDir string,
	args ...string,
) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return CommandResult{
			ExitCode: 0,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
		}, nil
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return CommandResult{}, fmt.Errorf("git command canceled: %w", ctx.Err())
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return CommandResult{
			ExitCode: exitErr.ExitCode(),
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
		}, nil
	}

	return CommandResult{}, fmt.Errorf("run git command: %w", err)
}
