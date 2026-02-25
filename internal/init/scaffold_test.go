package ralphinit_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	ralphinit "github.com/shuymn/ralph/internal/init"
)

const configSchemaLine = "# yaml-language-server: $schema=" +
	"https://raw.githubusercontent.com/shuymn/ralph/main/schemas/config.schema.json"

func TestScaffoldCreatesTemplateFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	var stderr bytes.Buffer
	fixed := time.Date(2026, time.February, 23, 10, 0, 0, 0, time.UTC)

	if err := ralphinit.Scaffold(root, fixed, &stderr); err != nil {
		t.Fatalf("Scaffold returned error: %v", err)
	}

	expected := []string{
		filepath.Join(root, ".ralph", "config.yml"),
		filepath.Join(root, ".ralph", "prompt.md"),
		filepath.Join(root, ".ralph", "prd.json"),
		filepath.Join(root, ".ralph", "progress.md"),
	}

	for _, path := range expected {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s to exist: %v", path, err)
		}
	}

	configBytes, err := os.ReadFile(filepath.Join(root, ".ralph", "config.yml"))
	if err != nil {
		t.Fatalf("failed reading config.yml: %v", err)
	}

	if !strings.Contains(string(configBytes), configSchemaLine) {
		t.Fatalf("config.yml missing schema header")
	}

	progressBytes, err := os.ReadFile(filepath.Join(root, ".ralph", "progress.md"))
	if err != nil {
		t.Fatalf("failed reading progress.md: %v", err)
	}

	if !strings.Contains(string(progressBytes), "Started: 2026-02-23") {
		t.Fatalf("progress.md missing rendered date, got:\n%s", string(progressBytes))
	}

	prdBytes, err := os.ReadFile(filepath.Join(root, ".ralph", "prd.json"))
	if err != nil {
		t.Fatalf("failed reading prd.json: %v", err)
	}
	if !strings.Contains(string(prdBytes), `"branchName": "replace-with-branch-name"`) {
		t.Fatalf("prd.json missing branchName scaffold, got:\n%s", string(prdBytes))
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output for fresh init, got: %s", stderr.String())
	}

	schemaCopyPath := filepath.Join(root, ".ralph", "config.schema.json")
	if _, err := os.Stat(schemaCopyPath); !os.IsNotExist(err) {
		t.Fatalf(
			"scaffold must not create %q (schema artifact must stay at repository-level)",
			schemaCopyPath,
		)
	}
}

func TestScaffoldSkipsExistingFilesAndReportsToStderr(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ralphDir := filepath.Join(root, ".ralph")
	if err := os.MkdirAll(ralphDir, 0o755); err != nil {
		t.Fatalf("failed creating .ralph directory: %v", err)
	}

	configPath := filepath.Join(ralphDir, "config.yml")
	progressPath := filepath.Join(ralphDir, "progress.md")
	if err := os.WriteFile(configPath, []byte("existing-config\n"), 0o600); err != nil {
		t.Fatalf("failed writing existing config.yml: %v", err)
	}
	if err := os.WriteFile(progressPath, []byte("existing-progress\n"), 0o600); err != nil {
		t.Fatalf("failed writing existing progress.md: %v", err)
	}

	var stderr bytes.Buffer
	fixed := time.Date(2026, time.February, 23, 10, 0, 0, 0, time.UTC)

	if err := ralphinit.Scaffold(root, fixed, &stderr); err != nil {
		t.Fatalf("Scaffold returned error: %v", err)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed reading config.yml: %v", err)
	}
	if string(configBytes) != "existing-config\n" {
		t.Fatalf("expected existing config.yml to remain unchanged, got: %q", string(configBytes))
	}

	progressBytes, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed reading progress.md: %v", err)
	}
	if string(progressBytes) != "existing-progress\n" {
		t.Fatalf(
			"expected existing progress.md to remain unchanged, got: %q",
			string(progressBytes),
		)
	}

	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, ".ralph/config.yml") {
		t.Fatalf("stderr missing skipped config path, got: %s", stderrOutput)
	}
	if !strings.Contains(stderrOutput, ".ralph/progress.md") {
		t.Fatalf("stderr missing skipped progress path, got: %s", stderrOutput)
	}

	for _, created := range []string{"prompt.md", "prd.json"} {
		if _, err := os.Stat(filepath.Join(ralphDir, created)); err != nil {
			t.Fatalf("expected %s to be created: %v", created, err)
		}
	}

	schemaCopyPath := filepath.Join(ralphDir, "config.schema.json")
	if _, err := os.Stat(schemaCopyPath); !os.IsNotExist(err) {
		t.Fatalf(
			"scaffold must not create %q when some files already exist",
			schemaCopyPath,
		)
	}
}

func TestScaffoldDocumentationListsSchemaWorkflow(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepoRoot(t)
	agentsPath := filepath.Join(repoRoot, "AGENTS.md")
	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed reading AGENTS.md: %v", err)
	}

	text := string(content)
	requiredSnippets := []string{
		"`task schema`",
		"`task check`",
		"schema",
		"drift",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("AGENTS.md must include schema workflow snippet %q", snippet)
		}
	}
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}
