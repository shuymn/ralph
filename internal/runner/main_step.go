package ralphrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type mainResult struct {
	Success    bool
	OutputPath string
}

func runMainStep(
	ctx context.Context,
	agentCommand string,
	promptPath string,
	opts Options,
	tracker *tmpTracker,
) (mainResult, error) {
	promptFile, err := os.Open(promptPath)
	if err != nil {
		return mainResult{}, fmt.Errorf("open prompt: %w", err)
	}
	defer func() {
		_ = promptFile.Close()
	}()

	tmpFile, err := tracker.create(opts.TempDir)
	if err != nil {
		return mainResult{}, fmt.Errorf("create main output tmpfile: %w", err)
	}

	result := mainResult{OutputPath: tmpFile.Name()}

	cmd := exec.CommandContext(ctx, "sh", "-c", agentCommand)
	cmd.Dir = opts.WorkingDir
	cmd.Stdin = promptFile
	cmd.Stdout = tmpFile
	cmd.Stderr = opts.Stderr

	runErr := cmd.Run()
	closeErr := tmpFile.Close()
	if closeErr != nil {
		tracker.cleanup(result.OutputPath)
		return mainResult{}, fmt.Errorf("close main output tmpfile: %w", closeErr)
	}

	if runErr != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return result, fmt.Errorf("main command canceled: %w", ctx.Err())
		}
		result.Success = false
		return result, nil
	}

	result.Success = true
	return result, nil
}

func runStepCommand(ctx context.Context, command string, opts Options) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = opts.WorkingDir
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("step command canceled: %w", ctx.Err())
		}
		return fmt.Errorf("run step command: %w", err)
	}

	return nil
}
