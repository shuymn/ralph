package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

const (
	shellCommand = "sh"
	shellFlag    = "-c"
)

type mainStepResult struct {
	Success bool
	Output  trackedTmpFile
	Err     error
}

func runMainStep(
	ctx context.Context,
	workDir string,
	promptPath string,
	command string,
	stderr io.Writer,
) mainStepResult {
	outputFile, tracked, err := createTrackedTmpFile(workDir)
	if err != nil {
		return mainStepResult{Err: fmt.Errorf("prepare main output: %w", err)}
	}
	defer outputFile.Close()

	promptFile, err := os.Open(promptPath)
	if err != nil {
		return mainStepResult{
			Output: tracked,
			Err:    fmt.Errorf("open prompt file: %w", err),
		}
	}
	defer promptFile.Close()

	cmd := exec.CommandContext(ctx, shellCommand, shellFlag, command)
	cmd.Dir = workDir
	cmd.Stdin = promptFile
	cmd.Stdout = outputFile
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return mainStepResult{
			Output: tracked,
			Err:    fmt.Errorf("run main command: %w", err),
		}
	}

	return mainStepResult{Success: true, Output: tracked}
}
