package ralphconfig

const (
	SupportedVersion = "1"

	DefaultAgentMaxIterations = 60
	DefaultAgentSleepSeconds  = 5

	DefaultCompletionStrategy  = "tail_match"
	DefaultCompletionSignal    = "<promise>COMPLETE</promise>"
	DefaultCompletionTailLines = 20

	DefaultGitCommitMode  = "split"
	DefaultFallbackCommit = "feat: implement task (auto-commit)"

	DefaultStepIf     = "success()"
	DefaultStepOnFail = "stop_loop"
)

type Config struct {
	Version    string     `yaml:"version"`
	Agent      Agent      `yaml:"agent"`
	Completion Completion `yaml:"completion"`
	Git        Git        `yaml:"git"`
	Phases     Phases     `yaml:"phases"`
}

type Agent struct {
	Command       string `yaml:"command"`
	MaxIterations int    `yaml:"max_iterations"` //nolint:tagliatelle // External schema key uses snake_case.
	SleepSeconds  int    `yaml:"sleep_seconds"`  //nolint:tagliatelle // External schema key uses snake_case.
}

type Completion struct {
	Strategy  string `yaml:"strategy"`
	Signal    string `yaml:"signal"`
	TailLines int    `yaml:"tail_lines"` //nolint:tagliatelle // External schema key uses snake_case.
}

type Git struct {
	Commit          string `yaml:"commit"`
	FallbackMessage string `yaml:"fallback_message"` //nolint:tagliatelle // External schema key uses snake_case.
}

type Phases struct {
	Pre  Phase `yaml:"pre"`
	Post Phase `yaml:"post"`
}

type Phase struct {
	Steps []Step `yaml:"steps"`
}

type Step struct {
	Name   string `yaml:"name"`
	Run    string `yaml:"run"`
	Uses   string `yaml:"uses"`
	If     string `yaml:"if"`
	OnFail string `yaml:"on_fail"` //nolint:tagliatelle // External schema key uses snake_case.
}
