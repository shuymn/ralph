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

func RunRun(root string, stdout, stderr io.Writer) int {
	configPath := filepath.Join(root, ".ralph", "config.yml")
	cfg, err := ralphconfig.Load(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ralph run failed: %v\n", err)
		return errorExitCode(err, 1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return ralphrunner.Run(ctx, cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     stdout,
		Stderr:     stderr,
	})
}

func errorExitCode(err error, fallback int) int {
	var exitCoder interface{ ExitCode() int }
	if errors.As(err, &exitCoder) {
		return exitCoder.ExitCode()
	}
	return fallback
}
