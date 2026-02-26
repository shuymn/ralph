package ralphconfig

const (
	SupportedVersion = "1"

	DefaultAgentMaxIterations = 60
	DefaultAgentSleepSeconds  = 5

	DefaultRunCompletionStrategy    = "tail_match"
	DefaultReviewCompletionStrategy = "review_convergence"
	DefaultCompletionSignal         = "<promise>COMPLETE</promise>"
	DefaultRunCompletionTailLines   = 20

	DefaultReviewMinReviews   = 3
	DefaultReviewMaxReviews   = 10
	DefaultReviewJudgeEvery   = 2
	DefaultReviewStableRounds = 2

	DefaultGitCommitMode  = "split"
	DefaultFallbackCommit = "feat: implement task (auto-commit)"

	DefaultStepIf     = "success()"
	DefaultStepOnFail = "stop_loop"
)

type Config struct {
	Version    string     `json:"version"             yaml:"version"`
	Agent      Agent      `json:"agent"               yaml:"agent"`
	Completion Completion `json:"completion,omitzero" yaml:"completion"`
	Git        Git        `json:"git,omitzero"        yaml:"git"`
	Phases     Phases     `json:"phases,omitzero"     yaml:"phases"`
}

type Agent struct {
	RunCommand    string `json:"run_command"              yaml:"run_command"`    //nolint:tagliatelle // External schema key uses snake_case.
	ReviewCommand string `json:"review_command,omitempty" yaml:"review_command"` //nolint:tagliatelle // External schema key uses snake_case.
	JudgeCommand  string `json:"judge_command,omitempty"  yaml:"judge_command"`  //nolint:tagliatelle // External schema key uses snake_case.
	MaxIterations int    `json:"max_iterations,omitempty" yaml:"max_iterations"` //nolint:tagliatelle // External schema key uses snake_case.
	SleepSeconds  int    `json:"sleep_seconds,omitempty"  yaml:"sleep_seconds"`  //nolint:tagliatelle // External schema key uses snake_case.
}

type Completion struct {
	Run    RunCompletionProfile    `json:"run,omitzero"    yaml:"run"`
	Review ReviewCompletionProfile `json:"review,omitzero" yaml:"review"`
}

type RunCompletionProfile struct {
	Strategy          string `json:"strategy,omitempty"   yaml:"strategy"`
	Signal            string `json:"signal,omitempty"     yaml:"signal"`
	TailLines         int    `json:"tail_lines,omitempty" yaml:"tail_lines"` //nolint:tagliatelle // External schema key uses snake_case.
	tailLinesExplicit bool   `json:"-"                    yaml:"-"`
}

type ReviewCompletionProfile struct {
	Strategy          string                `json:"strategy,omitempty"          yaml:"strategy"`
	Signal            string                `json:"signal,omitempty"            yaml:"signal"`
	ReviewConvergence ReviewConvergenceMode `json:"review_convergence,omitzero" yaml:"review_convergence"` //nolint:tagliatelle // External schema key uses snake_case.
}

type ReviewConvergenceMode struct {
	MinReviews   int `json:"min_reviews,omitempty"   yaml:"min_reviews"`   //nolint:tagliatelle // External schema key uses snake_case.
	MaxReviews   int `json:"max_reviews,omitempty"   yaml:"max_reviews"`   //nolint:tagliatelle // External schema key uses snake_case.
	JudgeEvery   int `json:"judge_every,omitempty"   yaml:"judge_every"`   //nolint:tagliatelle // External schema key uses snake_case.
	StableRounds int `json:"stable_rounds,omitempty" yaml:"stable_rounds"` //nolint:tagliatelle // External schema key uses snake_case.

	minReviewsExplicit   bool `json:"-" yaml:"-"`
	maxReviewsExplicit   bool `json:"-" yaml:"-"`
	judgeEveryExplicit   bool `json:"-" yaml:"-"`
	stableRoundsExplicit bool `json:"-" yaml:"-"`
}

type Git struct {
	Commit            string `json:"commit,omitempty"               yaml:"commit"`
	FallbackMessage   string `json:"fallback_message,omitempty"     yaml:"fallback_message"`     //nolint:tagliatelle // External schema key uses snake_case.
	FallbackNoGPGSign bool   `json:"fallback_no_gpg_sign,omitempty" yaml:"fallback_no_gpg_sign"` //nolint:tagliatelle // External schema key uses snake_case.
}

type Phases struct {
	Pre  Phase `json:"pre,omitzero"  yaml:"pre"`
	Post Phase `json:"post,omitzero" yaml:"post"`
}

type Phase struct {
	Steps []Step `json:"steps,omitempty" yaml:"steps"`
}

type Step struct {
	Name   string `json:"name"              yaml:"name"`
	Run    string `json:"run,omitempty"     yaml:"run"`
	Uses   string `json:"uses,omitempty"    yaml:"uses"`
	If     string `json:"if,omitempty"      yaml:"if"`
	OnFail string `json:"on_fail,omitempty" yaml:"on_fail"` //nolint:tagliatelle // External schema key uses snake_case.
}
