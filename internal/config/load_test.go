package ralphconfig_test

import (
	"errors"
	"testing"

	ralphconfig "github.com/shuymn/ralph/internal/config"
)

func TestLoadBytesAppliesStepDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
git:
  commit: split
phases:
  pre:
    steps: []
  post:
    steps:
      - name: run_tests
        run: "go test ./..."
`))
	if err != nil {
		t.Fatalf("LoadBytes returned error: %v", err)
	}

	if cfg.Phases.Post.Steps[0].If != ralphconfig.DefaultStepIf {
		t.Fatalf(
			"expected default if=%q, got %q",
			ralphconfig.DefaultStepIf,
			cfg.Phases.Post.Steps[0].If,
		)
	}

	if cfg.Phases.Post.Steps[0].OnFail != ralphconfig.DefaultStepOnFail {
		t.Fatalf(
			"expected default on_fail=%q, got %q",
			ralphconfig.DefaultStepOnFail,
			cfg.Phases.Post.Steps[0].OnFail,
		)
	}
}

func TestLoadBytesRejectsInvalidStepShape(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
git:
  commit: split
phases:
  pre:
    steps:
      - name: bad_step
        run: "echo pre"
        uses: auto_commit
  post:
    steps: []
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigStepShape)
}

func TestLoadBytesRejectsMissingStepName(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
git:
  commit: split
phases:
  pre:
    steps:
      - run: "echo pre"
  post:
    steps: []
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigStepName)
}

func TestLoadBytesRejectsUnsupportedBuiltin(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
git:
  commit: split
phases:
  pre:
    steps: []
  post:
    steps:
      - name: bad_builtin
        uses: not_supported
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigUnsupported)
}

func TestLoadBytesRejectsInvalidOnFail(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
git:
  commit: split
phases:
  pre:
    steps: []
  post:
    steps:
      - name: bad_on_fail
        run: "echo post"
        on_fail: retry
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigOnFail)
}

func TestLoadBytesRejectsInvalidCommitMode(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
git:
  commit: grouped
phases:
  pre:
    steps: []
  post:
    steps: []
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigGitCommit)
}

func TestLoadBytesAppliesFallbackNoGPGSign(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		gitBlock  string
		wantValue bool
	}{
		{
			name: "default false when omitted",
			gitBlock: `
git:
  commit: split
`,
			wantValue: false,
		},
		{
			name: "explicit true",
			gitBlock: `
git:
  commit: split
  fallback_no_gpg_sign: true
`,
			wantValue: true,
		},
		{
			name: "explicit false",
			gitBlock: `
git:
  commit: split
  fallback_no_gpg_sign: false
`,
			wantValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			content := `
version: "1"
agent:
  command: "echo hello"
` + tc.gitBlock + `
phases:
  pre:
    steps: []
  post:
    steps: []
`
			cfg, err := ralphconfig.LoadBytes([]byte(content))
			if err != nil {
				t.Fatalf("LoadBytes returned error: %v", err)
			}
			if cfg.Git.FallbackNoGPGSign != tc.wantValue {
				t.Fatalf(
					"git.fallback_no_gpg_sign=%t, want %t",
					cfg.Git.FallbackNoGPGSign,
					tc.wantValue,
				)
			}
		})
	}
}

func TestLoadBytesRejectsUnsupportedIfExpression(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
git:
  commit: split
phases:
  pre:
    steps:
      - name: bad_if
        run: "echo pre"
        if: steps.main.success
  post:
    steps: []
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigParse)
}

func TestLoadBytesRejectsUnknownTopLevelField(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
git:
  commit: split
phases:
  pre:
    steps: []
  post:
    steps: []
unknown_top_level: true
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigParse)
}

func TestLoadBytesRejectsUnknownNestedField(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
  unknown_nested: 1
git:
  commit: split
phases:
  pre:
    steps: []
  post:
    steps: []
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigParse)
}

func assertErrorCode(t *testing.T, err error, wantCode string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error code %q, got nil", wantCode)
	}

	var coded interface {
		Code() string
		ExitCode() int
	}
	if !errors.As(err, &coded) {
		t.Fatalf("expected coded error, got: %T %v", err, err)
	}

	if coded.Code() != wantCode {
		t.Fatalf("unexpected error code: want=%q got=%q", wantCode, coded.Code())
	}
	if coded.ExitCode() != 22 {
		t.Fatalf("unexpected exit code: want=22 got=%d", coded.ExitCode())
	}
}
