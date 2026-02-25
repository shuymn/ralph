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
	Version    string     `json:"version"             yaml:"version"`
	Agent      Agent      `json:"agent"               yaml:"agent"`
	Completion Completion `json:"completion,omitzero" yaml:"completion"`
	Git        Git        `json:"git,omitzero"        yaml:"git"`
	Phases     Phases     `json:"phases,omitzero"     yaml:"phases"`
}

type Agent struct {
	Command       string `json:"command"                  yaml:"command"`
	MaxIterations int    `json:"max_iterations,omitempty" yaml:"max_iterations"` //nolint:tagliatelle // External schema key uses snake_case.
	SleepSeconds  int    `json:"sleep_seconds,omitempty"  yaml:"sleep_seconds"`  //nolint:tagliatelle // External schema key uses snake_case.
}

type Completion struct {
	Strategy  string `json:"strategy,omitempty"   yaml:"strategy"`
	Signal    string `json:"signal,omitempty"     yaml:"signal"`
	TailLines int    `json:"tail_lines,omitempty" yaml:"tail_lines"` //nolint:tagliatelle // External schema key uses snake_case.
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
