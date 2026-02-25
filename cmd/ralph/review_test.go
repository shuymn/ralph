package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ralphconfig "github.com/shuymn/ralph/internal/config"
)

func TestReviewUsesSharedConfigPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".ralph"), 0o755); err != nil {
		t.Fatalf("mkdir .ralph: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, ".ralph", "config.yml"),
		[]byte("version: ["),
		0o600,
	); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stderr bytes.Buffer
	exitCode := RunReview(root, io.Discard, &stderr)

	if exitCode != ralphconfig.ExitCodeValidation {
		t.Fatalf(
			"expected validation exit code %d, got %d",
			ralphconfig.ExitCodeValidation,
			exitCode,
		)
	}
	if !strings.Contains(stderr.String(), "ralph review failed: parse config YAML") {
		t.Fatalf("expected review command parse error, got stderr: %q", stderr.String())
	}
}

func TestReviewRejectsUnsupportedStrategy(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".ralph"), 0o755); err != nil {
		t.Fatalf("mkdir .ralph: %v", err)
	}

	config := `version: "1"
agent:
  run_command: "echo hi"
completion:
  run:
    strategy: tail_match
  review:
    strategy: tail_match
`
	if err := os.WriteFile(
		filepath.Join(root, ".ralph", "config.yml"),
		[]byte(config),
		0o600,
	); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stderr bytes.Buffer
	exitCode := RunReview(root, io.Discard, &stderr)

	if exitCode != ralphconfig.ExitCodeValidation {
		t.Fatalf(
			"expected validation exit code %d, got %d",
			ralphconfig.ExitCodeValidation,
			exitCode,
		)
	}
	if !strings.Contains(
		stderr.String(),
		"ralph review failed: completion.review.strategy must be review_convergence",
	) {
		t.Fatalf("expected strategy guard error, got stderr: %q", stderr.String())
	}
}
