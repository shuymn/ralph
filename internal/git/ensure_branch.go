package ralphgit

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const defaultBaseBranch = "main"

var (
	errEnsureBranchCommandFailed = errors.New("git ensure-branch command failed")
	errCurrentBranchCommand      = errors.New("git current-branch command failed")
	errBranchLookupCommand       = errors.New("git branch-lookup command failed")
)

type EnsureBranchOptions struct {
	WorkingDir string
	BranchName string
	BaseBranch string
	Runner     CommandRunner
}

// EnsureBranch moves to the configured branch.
// If the branch does not exist, it is created from BaseBranch.
func EnsureBranch(ctx context.Context, opts EnsureBranchOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}

	opts = normalizeEnsureBranchOptions(opts)
	branchName := strings.TrimSpace(opts.BranchName)
	if branchName == "" {
		return fmt.Errorf("%w: branch name is required", errEnsureBranchCommandFailed)
	}

	currentBranch, err := readCurrentBranch(ctx, opts)
	if err != nil {
		return err
	}
	if currentBranch == branchName {
		return nil
	}

	exists, err := localBranchExists(ctx, opts, branchName)
	if err != nil {
		return err
	}
	if exists {
		return runGitBranchCommand(ctx, opts, "switch", branchName)
	}

	baseBranch := strings.TrimSpace(opts.BaseBranch)
	if branchName == baseBranch {
		return runGitBranchCommand(ctx, opts, "switch", "-c", branchName)
	}

	baseExists, err := localBranchExists(ctx, opts, baseBranch)
	if err != nil {
		return err
	}
	if baseExists {
		return runGitBranchCommand(ctx, opts, "switch", "-c", branchName, baseBranch)
	}

	return runGitBranchCommand(ctx, opts, "switch", "-c", branchName)
}

func normalizeEnsureBranchOptions(opts EnsureBranchOptions) EnsureBranchOptions {
	if strings.TrimSpace(opts.BaseBranch) == "" {
		opts.BaseBranch = defaultBaseBranch
	}
	if opts.Runner == nil {
		opts.Runner = cliRunner{}
	}
	return opts
}

func readCurrentBranch(ctx context.Context, opts EnsureBranchOptions) (string, error) {
	result, err := opts.Runner.Run(ctx, opts.WorkingDir, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("%w: %w", errCurrentBranchCommand, err)
	}
	if result.ExitCode != 0 {
		return "", newBranchCommandExitError(errCurrentBranchCommand, result.ExitCode, result)
	}

	return strings.TrimSpace(result.Stdout), nil
}

func localBranchExists(ctx context.Context, opts EnsureBranchOptions, branch string) (bool, error) {
	result, err := opts.Runner.Run(
		ctx,
		opts.WorkingDir,
		"show-ref",
		"--verify",
		"--quiet",
		"refs/heads/"+branch,
	)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errBranchLookupCommand, err)
	}

	switch result.ExitCode {
	case 0:
		return true, nil
	case 1:
		return false, nil
	default:
		return false, newBranchCommandExitError(errBranchLookupCommand, result.ExitCode, result)
	}
}

func runGitBranchCommand(
	ctx context.Context,
	opts EnsureBranchOptions,
	args ...string,
) error {
	result, err := opts.Runner.Run(ctx, opts.WorkingDir, args...)
	if err != nil {
		return fmt.Errorf(
			"%w: command=%s: %w",
			errEnsureBranchCommandFailed,
			strings.Join(args, " "),
			err,
		)
	}
	if result.ExitCode == 0 {
		return nil
	}
	return newBranchCommandExitErrorWithCommand(
		errEnsureBranchCommandFailed,
		result.ExitCode,
		result,
		args,
	)
}

func newBranchCommandExitError(
	baseErr error,
	exitCode int,
	result CommandResult,
) error {
	return newBranchCommandExitErrorWithCommand(baseErr, exitCode, result, nil)
}

func newBranchCommandExitErrorWithCommand(
	baseErr error,
	exitCode int,
	result CommandResult,
	args []string,
) error {
	reason := strings.TrimSpace(result.Stderr)
	if reason == "" {
		reason = strings.TrimSpace(result.Stdout)
	}

	if len(args) > 0 {
		if reason == "" {
			return fmt.Errorf(
				"%w: command=%s exit_code=%d",
				baseErr,
				strings.Join(args, " "),
				exitCode,
			)
		}
		return fmt.Errorf(
			"%w: command=%s exit_code=%d reason=%s",
			baseErr,
			strings.Join(args, " "),
			exitCode,
			reason,
		)
	}

	if reason == "" {
		return fmt.Errorf("%w: exit_code=%d", baseErr, exitCode)
	}
	return fmt.Errorf("%w: exit_code=%d reason=%s", baseErr, exitCode, reason)
}
