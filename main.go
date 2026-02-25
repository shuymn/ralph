package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphinit "github.com/shuymn/ralph/internal/init"
	ralphrunner "github.com/shuymn/ralph/internal/runner"
)

const (
	usage       = "usage: ralph <init|run [--dry-run]|review [--dry-run]>"
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2

	commandWithOptionalFlagArgs = 2
)

func main() {
	exitCode := run(os.Args[1:])
	os.Exit(exitCode)
}

func run(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, usage)
		return exitUsage
	}

	switch args[0] {
	case "init":
		if len(args) != 1 {
			_, _ = fmt.Fprintln(os.Stderr, usage)
			return exitUsage
		}
		if err := ralphinit.Scaffold(".", time.Now(), os.Stderr); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "ralph init failed: %v\n", err)
			return exitFailure
		}
		return exitOK
	case "run":
		dryRun, ok := parseDryRun(args)
		if !ok {
			_, _ = fmt.Fprintln(os.Stderr, usage)
			return exitUsage
		}
		return runLoop(".", "run", dryRun)
	case "review":
		dryRun, ok := parseDryRun(args)
		if !ok {
			_, _ = fmt.Fprintln(os.Stderr, usage)
			return exitUsage
		}
		return runLoop(".", "review", dryRun)
	default:
		_, _ = fmt.Fprintln(os.Stderr, usage)
		return exitUsage
	}
}

func parseDryRun(args []string) (bool, bool) {
	if len(args) > commandWithOptionalFlagArgs {
		return false, false
	}
	if len(args) == commandWithOptionalFlagArgs {
		if args[1] != "--dry-run" {
			return false, false
		}
		return true, true
	}
	return false, true
}

func runLoop(root, command string, dryRun bool) int {
	configPath := filepath.Join(root, ".ralph", "config.yml")
	cfg, err := ralphconfig.Load(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ralph %s failed: %v\n", command, err)
		return errorExitCode(err, exitFailure)
	}
	if command == "review" &&
		cfg.Completion.Review.Strategy != ralphconfig.DefaultReviewCompletionStrategy {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"ralph review failed: completion.review.strategy must be %s\n",
			ralphconfig.DefaultReviewCompletionStrategy,
		)
		return ralphconfig.ExitCodeValidation
	}

	if dryRun {
		return ralphrunner.DryRun(cfg, ralphrunner.Options{
			WorkingDir: root,
			Stdout:     os.Stdout,
			Stderr:     os.Stderr,
		})
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return ralphrunner.Run(ctx, cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	})
}

func errorExitCode(err error, fallback int) int {
	var exitCoder interface{ ExitCode() int }
	if errors.As(err, &exitCoder) {
		return exitCoder.ExitCode()
	}
	return fallback
}
