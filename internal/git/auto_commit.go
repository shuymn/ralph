package ralphgit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ralphprd "github.com/shuymn/ralph/internal/prd"
)

const (
	CommitModeSplit    = "split"
	CommitModeTogether = "together"

	ralphOnlyPath        = ".ralph/"
	noGPGSignFlag        = "--no-gpg-sign"
	ralphCommitMessage   = "chore(ralph): mark %s complete in PRD and progress"
	commitNoSignOverhead = 2
	noStagedChangesCode  = 0
	hasStagedChangesCode = 1
)

var (
	errUnsupportedCommitMode = errors.New("unsupported git commit mode")
	errGitDiffStagedFailed   = errors.New("git diff --cached --quiet --exit-code failed")
	errGitDiffNameOnlyFailed = errors.New("git diff --cached --name-only failed")
	errGitCommandFailed      = errors.New("git command failed")
	errReadCommitMessageFile = errors.New("read commit message file")
	errRemoveCommitMsgFile   = errors.New("remove commit message file")
	errInvalidCommitMsgPath  = errors.New("invalid commit message path")
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
	FallbackNoGPGSign bool
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
	source, err := resolveCommitMessageSource(opts.CommitMessagePath, opts.FallbackMessage)
	if err != nil {
		return err
	}

	commitMsgPathspec, err := commitMessagePathspec(opts.WorkingDir, opts.CommitMessagePath)
	if err != nil {
		return err
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
	if otherChanged {
		if err := commitWithSource(ctx, opts, source); err != nil {
			return err
		}
	}

	if err := runGitExpectSuccess(ctx, opts, "add", "-A", ralphOnlyPath); err != nil {
		return err
	}
	if err := unstagePathIfStaged(ctx, opts, commitMsgPathspec); err != nil {
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
		if err := runGitCommit(ctx, opts, "-m", message); err != nil {
			return err
		}
	}

	return removeCommitMessageFile(opts.CommitMessagePath)
}

func autoCommitTogether(ctx context.Context, opts Options) error {
	source, err := resolveCommitMessageSource(opts.CommitMessagePath, opts.FallbackMessage)
	if err != nil {
		return err
	}

	commitMsgPathspec, err := commitMessagePathspec(opts.WorkingDir, opts.CommitMessagePath)
	if err != nil {
		return err
	}

	if err := runGitExpectSuccess(ctx, opts, "add", "-A"); err != nil {
		return err
	}
	if err := unstagePathIfStaged(ctx, opts, commitMsgPathspec); err != nil {
		return err
	}

	stagedChanges, err := hasStagedDiff(ctx, opts)
	if err != nil {
		return err
	}
	if !stagedChanges {
		return removeCommitMessageFile(opts.CommitMessagePath)
	}

	if err := commitWithSource(ctx, opts, source); err != nil {
		return err
	}
	return removeCommitMessageFile(opts.CommitMessagePath)
}

func commitWithSource(
	ctx context.Context,
	opts Options,
	source commitMessageSource,
) error {
	if source.useFile {
		return runGitCommit(ctx, opts, "-F", source.value)
	}
	return runGitCommit(ctx, opts, "-m", source.value)
}

func runGitCommit(ctx context.Context, opts Options, args ...string) error {
	commitArgs := make([]string, 0, len(args)+1)
	commitArgs = append(commitArgs, "commit")
	commitArgs = append(commitArgs, args...)

	err := runGitExpectSuccess(ctx, opts, commitArgs...)
	if err == nil {
		return nil
	}
	if !opts.FallbackNoGPGSign || !isGPGSignFailure(err) {
		return err
	}

	retryArgs := make([]string, 0, len(args)+commitNoSignOverhead)
	retryArgs = append(retryArgs, "commit", noGPGSignFlag)
	retryArgs = append(retryArgs, args...)
	if retryErr := runGitExpectSuccess(ctx, opts, retryArgs...); retryErr != nil {
		return fmt.Errorf("retry commit without gpg-sign: %w", retryErr)
	}
	return nil
}

func isGPGSignFailure(err error) bool {
	reason := strings.ToLower(gitCommandFailureReason(err))
	for _, marker := range []string{
		"gpg failed to sign the data",
		"gpg: signing failed",
		"failed to sign the data",
	} {
		if strings.Contains(reason, marker) {
			return true
		}
	}
	return false
}

func gitCommandFailureReason(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	const reasonPrefix = " reason="
	idx := strings.LastIndex(msg, reasonPrefix)
	if idx == -1 {
		return msg
	}
	return msg[idx+len(reasonPrefix):]
}

func unstagePathIfStaged(ctx context.Context, opts Options, pathspec string) error {
	staged, err := hasStagedDiffForPath(ctx, opts, pathspec)
	if err != nil {
		return err
	}
	if !staged {
		return nil
	}
	return runGitExpectSuccess(ctx, opts, "restore", "--staged", pathspec)
}

func hasStagedDiffForPath(ctx context.Context, opts Options, pathspec string) (bool, error) {
	result, err := opts.Runner.Run(
		ctx,
		opts.WorkingDir,
		"diff",
		"--cached",
		"--name-only",
		"--",
		pathspec,
	)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errGitDiffNameOnlyFailed, err)
	}
	if result.ExitCode != 0 {
		reason := strings.TrimSpace(result.Stderr)
		if reason == "" {
			reason = strings.TrimSpace(result.Stdout)
		}
		if reason == "" {
			return false, fmt.Errorf(
				"%w: exit_code=%d",
				errGitDiffNameOnlyFailed,
				result.ExitCode,
			)
		}
		return false, fmt.Errorf(
			"%w: exit_code=%d reason=%s",
			errGitDiffNameOnlyFailed,
			result.ExitCode,
			reason,
		)
	}
	return strings.TrimSpace(result.Stdout) != "", nil
}

func commitMessagePathspec(workingDir, commitMessagePath string) (string, error) {
	path := strings.TrimSpace(commitMessagePath)
	if path == "" {
		return "", fmt.Errorf("%w: commit message path is empty", errInvalidCommitMsgPath)
	}

	relPath, err := filepath.Rel(workingDir, path)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errInvalidCommitMsgPath, err)
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf(
			"%w: path=%q working_dir=%q",
			errInvalidCommitMsgPath,
			commitMessagePath,
			workingDir,
		)
	}
	return filepath.ToSlash(relPath), nil
}

func removeCommitMessageFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	err := os.Remove(path)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("%w: %w", errRemoveCommitMsgFile, err)
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
