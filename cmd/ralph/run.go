package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/signal"
	"path/filepath"
	"syscall"

	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphrunner "github.com/shuymn/ralph/internal/runner"
)

const (
	commandRun    = "run"
	commandReview = "review"
)

func RunRun(root string, stdout, stderr io.Writer) int {
	return runRun(root, stdout, stderr, false)
}

func RunRunDry(root string, stdout, stderr io.Writer) int {
	return runRun(root, stdout, stderr, true)
}

func runRun(root string, stdout, stderr io.Writer, dryRun bool) int {
	return runCommand(root, stdout, stderr, commandRun, dryRun)
}

func runCommand(root string, stdout, stderr io.Writer, command string, dryRun bool) int {
	configPath := filepath.Join(root, ".ralph", "config.yml")
	cfg, err := ralphconfig.Load(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ralph %s failed: %v\n", command, err)
		return errorExitCode(err, 1)
	}
	mode := resolveCommandMode(command)
	if command == commandReview &&
		cfg.Completion.Review.Strategy != ralphconfig.DefaultReviewCompletionStrategy {
		_, _ = fmt.Fprintf(
			stderr,
			"ralph review failed: completion.review.strategy must be %s\n",
			ralphconfig.DefaultReviewCompletionStrategy,
		)
		return ralphconfig.ExitCodeValidation
	}

	if dryRun {
		return ralphrunner.DryRun(cfg, ralphrunner.Options{
			WorkingDir: root,
			Stdout:     stdout,
			Stderr:     stderr,
			Mode:       mode,
		})
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return ralphrunner.Run(ctx, cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     stdout,
		Stderr:     stderr,
		Mode:       mode,
	})
}

func resolveCommandMode(command string) ralphrunner.Mode {
	if command == commandReview {
		return ralphrunner.ModeReview
	}
	return ralphrunner.ModeRun
}

func errorExitCode(err error, fallback int) int {
	var exitCoder interface{ ExitCode() int }
	if errors.As(err, &exitCoder) {
		return exitCoder.ExitCode()
	}
	return fallback
}
