package ralphconfig_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	ralphconfig "github.com/shuymn/ralph/internal/config"
)

func TestLoadBytesAppliesStepDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  run_command: "echo hello"
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
  run_command: "echo hello"
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
  run_command: "echo hello"
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
  run_command: "echo hello"
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
  run_command: "echo hello"
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
  run_command: "echo hello"
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

func TestLoadBytesRejectsMissingRunCommand(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		agentBlock string
	}{
		{
			name:       "missing agent block",
			agentBlock: "",
		},
		{
			name: "missing run_command field",
			agentBlock: `
agent:
  sleep_seconds: 5
`,
		},
		{
			name: "blank run_command value",
			agentBlock: `
agent:
  run_command: "   "
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			content := `
version: "1"
` + tc.agentBlock + `
git:
  commit: split
phases:
  pre:
    steps: []
  post:
    steps: []
`

			_, err := ralphconfig.LoadBytes([]byte(content))
			assertErrorCode(t, err, ralphconfig.ErrCodeConfigParse)
			assertErrorContains(t, err, "agent.run_command is required")
		})
	}
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
  run_command: "echo hello"
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
  run_command: "echo hello"
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
  run_command: "echo hello"
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
  run_command: "echo hello"
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

func TestLoadBytesAcceptsRunAndReviewCompletionProfiles(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  run_command: "echo hello"
completion:
  run:
    strategy: tail_match
  review:
    strategy: review_convergence
`))
	if err != nil {
		t.Fatalf("LoadBytes returned error: %v", err)
	}
}

func TestLoadBytesRejectsLegacyAgentCommandKey(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  command: "echo hello"
completion:
  run:
    strategy: tail_match
  review:
    strategy: review_convergence
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigParse)
}

func TestLoadBytesRejectsLegacyCompletionShape(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  run_command: "echo hello"
completion:
  strategy: tail_match
  signal: "<promise>COMPLETE</promise>"
`))
	assertErrorCode(t, err, ralphconfig.ErrCodeConfigParse)
}

func TestLoadBytesRejectsInvalidRunStrategy(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  run_command: "echo hello"
completion:
  run:
    strategy: review_convergence
  review:
    strategy: review_convergence
`))
	assertErrorContains(t, err, "completion.run.strategy must be tail_match")
}

func TestLoadBytesRejectsInvalidReviewStrategy(t *testing.T) {
	t.Parallel()

	_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  run_command: "echo hello"
completion:
  run:
    strategy: tail_match
  review:
    strategy: tail_match
`))
	assertErrorContains(t, err, "completion.review.strategy must be review_convergence")
}

func TestLoadBytesRejectsNonPositiveTailLines(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		tailLines int
	}{
		{name: "negative tail_lines", tailLines: -1},
		{name: "zero tail_lines", tailLines: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := ralphconfig.LoadBytes([]byte(`
version: "1"
agent:
  run_command: "echo hello"
completion:
  run:
    strategy: tail_match
    tail_lines: ` + strconv.Itoa(tc.tailLines) + `
  review:
    strategy: review_convergence
`))
			assertErrorContains(t, err, "completion.run.tail_lines must be >= 1")
		})
	}
}

func TestLoadBytesAppliesReviewConvergencePartialDefaults(t *testing.T) {
	t.Parallel()
	runReviewConvergenceSingleFieldOverrideCases(t, "")
}

func TestLoadBytesReviewConvergenceSingleFieldOverrides(t *testing.T) {
	t.Parallel()
	runReviewConvergenceSingleFieldOverrideCases(t, "overrides ")
}

func TestLoadBytesRejectsInvalidReviewConvergenceBounds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "min_reviews must be positive",
			yaml: `
version: "1"
agent:
  run_command: "echo hello"
completion:
  run:
    strategy: tail_match
  review:
    strategy: review_convergence
    review_convergence:
      min_reviews: -1
      max_reviews: 10
      judge_every: 2
      stable_rounds: 2
`,
			wantErr: "completion.review.review_convergence.min_reviews must be >= 1",
		},
		{
			name: "max_reviews must be >= min_reviews",
			yaml: `
version: "1"
agent:
  run_command: "echo hello"
completion:
  run:
    strategy: tail_match
  review:
    strategy: review_convergence
    review_convergence:
      min_reviews: 4
      max_reviews: 3
      judge_every: 2
      stable_rounds: 2
`,
			wantErr: "completion.review.review_convergence.max_reviews must be >= min_reviews",
		},
		{
			name: "judge_every must be positive",
			yaml: `
version: "1"
agent:
  run_command: "echo hello"
completion:
  run:
    strategy: tail_match
  review:
    strategy: review_convergence
    review_convergence:
      min_reviews: 3
      max_reviews: 10
      judge_every: -1
      stable_rounds: 2
`,
			wantErr: "completion.review.review_convergence.judge_every must be >= 1",
		},
		{
			name: "stable_rounds must be positive",
			yaml: `
version: "1"
agent:
  run_command: "echo hello"
completion:
  run:
    strategy: tail_match
  review:
    strategy: review_convergence
    review_convergence:
      min_reviews: 3
      max_reviews: 10
      judge_every: 2
      stable_rounds: -1
`,
			wantErr: "completion.review.review_convergence.stable_rounds must be >= 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ralphconfig.LoadBytes([]byte(tc.yaml))
			assertErrorContains(t, err, tc.wantErr)
		})
	}
}

func assertReviewConvergenceValues(
	t *testing.T,
	reviewConvergence string,
	wantMinReviews int,
	wantMaxReviews int,
	wantJudgeEvery int,
	wantStableRounds int,
) {
	t.Helper()

	content := `
version: "1"
agent:
  run_command: "echo hello"
completion:
  run:
    strategy: tail_match
  review:
    strategy: review_convergence
    review_convergence:
` + reviewConvergence

	cfg, err := ralphconfig.LoadBytes([]byte(content))
	if err != nil {
		t.Fatalf("LoadBytes returned error: %v", err)
	}

	got := cfg.Completion.Review.ReviewConvergence
	if got.MinReviews != wantMinReviews {
		t.Fatalf("min_reviews=%d, want %d", got.MinReviews, wantMinReviews)
	}
	if got.MaxReviews != wantMaxReviews {
		t.Fatalf("max_reviews=%d, want %d", got.MaxReviews, wantMaxReviews)
	}
	if got.JudgeEvery != wantJudgeEvery {
		t.Fatalf("judge_every=%d, want %d", got.JudgeEvery, wantJudgeEvery)
	}
	if got.StableRounds != wantStableRounds {
		t.Fatalf("stable_rounds=%d, want %d", got.StableRounds, wantStableRounds)
	}
}

func runReviewConvergenceSingleFieldOverrideCases(t *testing.T, namePrefix string) {
	t.Helper()

	testCases := []struct {
		name              string
		reviewConvergence string
		wantMinReviews    int
		wantMaxReviews    int
		wantJudgeEvery    int
		wantStableRounds  int
	}{
		{
			name: "min_reviews only",
			reviewConvergence: `
      min_reviews: 4
`,
			wantMinReviews:   4,
			wantMaxReviews:   ralphconfig.DefaultReviewMaxReviews,
			wantJudgeEvery:   ralphconfig.DefaultReviewJudgeEvery,
			wantStableRounds: ralphconfig.DefaultReviewStableRounds,
		},
		{
			name: "max_reviews only",
			reviewConvergence: `
      max_reviews: 12
`,
			wantMinReviews:   ralphconfig.DefaultReviewMinReviews,
			wantMaxReviews:   12,
			wantJudgeEvery:   ralphconfig.DefaultReviewJudgeEvery,
			wantStableRounds: ralphconfig.DefaultReviewStableRounds,
		},
		{
			name: "judge_every only",
			reviewConvergence: `
      judge_every: 3
`,
			wantMinReviews:   ralphconfig.DefaultReviewMinReviews,
			wantMaxReviews:   ralphconfig.DefaultReviewMaxReviews,
			wantJudgeEvery:   3,
			wantStableRounds: ralphconfig.DefaultReviewStableRounds,
		},
		{
			name: "stable_rounds only",
			reviewConvergence: `
      stable_rounds: 4
`,
			wantMinReviews:   ralphconfig.DefaultReviewMinReviews,
			wantMaxReviews:   ralphconfig.DefaultReviewMaxReviews,
			wantJudgeEvery:   ralphconfig.DefaultReviewJudgeEvery,
			wantStableRounds: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(namePrefix+tc.name, func(t *testing.T) {
			t.Parallel()
			assertReviewConvergenceValues(
				t,
				tc.reviewConvergence,
				tc.wantMinReviews,
				tc.wantMaxReviews,
				tc.wantJudgeEvery,
				tc.wantStableRounds,
			)
		})
	}
}

func assertErrorContains(t *testing.T, err error, wantSubstring string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantSubstring)
	}
	if !strings.Contains(err.Error(), wantSubstring) {
		t.Fatalf("error %q must contain %q", err.Error(), wantSubstring)
	}
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
