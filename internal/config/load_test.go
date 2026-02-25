package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuymn/ralph/internal/config"
)

func TestLoadReturnsValidationErrorsForInvalidStepAndGitConfiguration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		content     string
		wantMessage string
	}{
		{
			name: "step cannot define both run and uses",
			content: validConfigYAML(`
  pre:
    steps:
      - name: invalid_both
        run: "echo pre"
        uses: auto_commit
`),
			wantMessage: "exactly one of run or uses",
		},
		{
			name: "step name is required",
			content: validConfigYAML(`
  pre:
    steps:
      - run: "echo pre"
`),
			wantMessage: "name is required",
		},
		{
			name: "unsupported builtin is rejected",
			content: validConfigYAML(`
  pre:
    steps:
      - name: unknown_builtin
        uses: custom_builtin
`),
			wantMessage: "unsupported builtin",
		},
		{
			name: "invalid on_fail is rejected",
			content: validConfigYAML(`
  pre:
    steps:
      - name: invalid_on_fail
        run: "echo pre"
        on_fail: halt
`),
			wantMessage: "invalid on_fail",
		},
		{
			name: "invalid git.commit is rejected",
			content: strings.TrimSpace(`
version: "1"

agent:
  command: "echo run"

git:
  commit: grouped
  fallback_message: "fallback"

phases:
  pre:
    steps: []
  post:
    steps: []
`),
			wantMessage: "invalid git.commit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			configPath := writeConfigFile(t, tc.content)
			_, err := config.Load(configPath)
			if err == nil {
				t.Fatalf("Load() error = nil, want validation error")
			}
			if !config.IsValidationError(err) {
				t.Fatalf("Load() error kind = %T (%v), want validation", err, err)
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("Load() error = %q, want substring %q", err.Error(), tc.wantMessage)
			}
		})
	}
}

func TestLoadAppliesStepDefaults(t *testing.T) {
	t.Parallel()

	configPath := writeConfigFile(t, validConfigYAML(`
  pre:
    steps:
      - name: fmt
        run: "task fmt"
`))

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Phases.Pre.Steps) != 1 {
		t.Fatalf("len(cfg.Phases.Pre.Steps) = %d, want 1", len(cfg.Phases.Pre.Steps))
	}

	step := cfg.Phases.Pre.Steps[0]
	if step.If != config.DefaultIfExpr {
		t.Fatalf("step.If = %q, want %q", step.If, config.DefaultIfExpr)
	}
	if step.OnFail != config.DefaultOnFail {
		t.Fatalf("step.OnFail = %q, want %q", step.OnFail, config.DefaultOnFail)
	}
}

func TestLoadReturnsParseErrorForMalformedYAML(t *testing.T) {
	t.Parallel()

	configPath := writeConfigFile(t, "version: [")
	_, err := config.Load(configPath)
	if err == nil {
		t.Fatalf("Load() error = nil, want parse error")
	}
	if !config.IsParseError(err) {
		t.Fatalf("Load() error kind = %T (%v), want parse", err, err)
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	rootDir := t.TempDir()
	configPath := filepath.Join(rootDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile(config.yml) error = %v", err)
	}
	return configPath
}

func validConfigYAML(phasesOverride string) string {
	return strings.TrimSpace(`
version: "1"

agent:
  command: "echo run"

git:
  commit: split
  fallback_message: "fallback"

phases:
` + phasesOverride + `
`)
}
