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
	usage       = "usage: ralph <init|run>"
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

func main() {
	exitCode := run(os.Args[1:])
	os.Exit(exitCode)
}

func run(args []string) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(os.Stderr, usage)
		return exitUsage
	}

	switch args[0] {
	case "init":
		if err := ralphinit.Scaffold(".", time.Now(), os.Stderr); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "ralph init failed: %v\n", err)
			return exitFailure
		}
		return exitOK
	case "run":
		return runLoop(".")
	default:
		_, _ = fmt.Fprintln(os.Stderr, usage)
		return exitUsage
	}
}

func runLoop(root string) int {
	configPath := filepath.Join(root, ".ralph", "config.yml")
	cfg, err := ralphconfig.Load(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ralph run failed: %v\n", err)
		return errorExitCode(err, exitFailure)
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
