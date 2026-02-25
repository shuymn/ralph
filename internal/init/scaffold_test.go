package ralphinit_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ralphinit "github.com/shuymn/ralph/internal/init"
)

func TestScaffoldCreatesExpectedFiles(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	var stderr bytes.Buffer

	err := ralphinit.Scaffold(ralphinit.Options{
		RootDir: rootDir,
		Now:     time.Date(2026, time.February, 25, 10, 0, 0, 0, time.UTC),
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	expectedPaths := []string{
		filepath.Join(rootDir, ".ralph", "config.yml"),
		filepath.Join(rootDir, ".ralph", "prompt.md"),
		filepath.Join(rootDir, ".ralph", "prd.json"),
		filepath.Join(rootDir, ".ralph", "progress.md"),
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("os.Stat(%q) error = %v", path, err)
		}
	}

	configBytes, err := os.ReadFile(filepath.Join(rootDir, ".ralph", "config.yml"))
	if err != nil {
		t.Fatalf("os.ReadFile(config.yml) error = %v", err)
	}

	const schemaHeader = "# yaml-language-server: $schema=https://raw.githubusercontent.com/shuymn/ralph/main/schemas/config.schema.json"
	if !strings.Contains(string(configBytes), schemaHeader) {
		t.Fatalf("config.yml does not contain schema header %q", schemaHeader)
	}

	progressBytes, err := os.ReadFile(filepath.Join(rootDir, ".ralph", "progress.md"))
	if err != nil {
		t.Fatalf("os.ReadFile(progress.md) error = %v", err)
	}

	if !strings.Contains(string(progressBytes), "Started: 2026-02-25") {
		t.Fatalf("progress.md does not contain rendered start date; got: %q", string(progressBytes))
	}
}

func TestScaffoldSkipsExistingFilesAndLogsToStderr(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	ralphDir := filepath.Join(rootDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	const existingConfig = "existing config"
	configPath := filepath.Join(ralphDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(existingConfig), 0o600); err != nil {
		t.Fatalf("os.WriteFile(config.yml) error = %v", err)
	}

	const existingProgress = "existing progress"
	progressPath := filepath.Join(ralphDir, "progress.md")
	if err := os.WriteFile(progressPath, []byte(existingProgress), 0o600); err != nil {
		t.Fatalf("os.WriteFile(progress.md) error = %v", err)
	}

	var stderr bytes.Buffer
	err := ralphinit.Scaffold(ralphinit.Options{
		RootDir: rootDir,
		Now:     time.Date(2026, time.February, 25, 10, 0, 0, 0, time.UTC),
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile(config.yml) error = %v", err)
	}
	if string(configBytes) != existingConfig {
		t.Fatalf("config.yml was overwritten: got %q", string(configBytes))
	}

	progressBytes, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("os.ReadFile(progress.md) error = %v", err)
	}
	if string(progressBytes) != existingProgress {
		t.Fatalf("progress.md was overwritten: got %q", string(progressBytes))
	}

	stderrText := stderr.String()
	if !strings.Contains(stderrText, ".ralph/config.yml") {
		t.Fatalf("stderr missing skip entry for config.yml: %q", stderrText)
	}
	if !strings.Contains(stderrText, ".ralph/progress.md") {
		t.Fatalf("stderr missing skip entry for progress.md: %q", stderrText)
	}

	if _, err := os.Stat(filepath.Join(ralphDir, "prompt.md")); err != nil {
		t.Fatalf("prompt.md not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ralphDir, "prd.json")); err != nil {
		t.Fatalf("prd.json not created: %v", err)
	}
}
