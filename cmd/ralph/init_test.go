package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunInitSuccess(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	var stderr bytes.Buffer
	fixedNow := func() time.Time {
		return time.Date(2026, time.February, 25, 0, 0, 0, 0, time.UTC)
	}

	if err := RunInit(root, &stderr, fixedNow); err != nil {
		t.Fatalf("RunInit returned error: %v", err)
	}

	expected := []string{
		filepath.Join(root, ".ralph", "config.yml"),
		filepath.Join(root, ".ralph", "prompt.run.md"),
		filepath.Join(root, ".ralph", "prompt.review.md"),
		filepath.Join(root, ".ralph", "prompt.judge.md"),
		filepath.Join(root, ".ralph", "prd.json"),
		filepath.Join(root, ".ralph", "progress.md"),
	}
	for _, path := range expected {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s to exist: %v", path, err)
		}
	}
}

func TestRunInitFailureWrapsScaffoldError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	blockingPath := filepath.Join(root, ".ralph")
	if err := os.WriteFile(
		blockingPath,
		[]byte("file blocks directory creation\n"),
		0o600,
	); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	err := RunInit(root, bytes.NewBuffer(nil), time.Now)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "init scaffold:") {
		t.Fatalf("expected wrapped scaffold error, got: %v", err)
	}
}
