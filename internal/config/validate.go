package ralphconfig

import (
	"fmt"
	"strings"

	ralphcondition "github.com/shuymn/ralph/internal/condition"
)

const (
	ExitCodeValidation = 22

	ErrCodeConfigParse       = "CONFIG_PARSE"
	ErrCodeConfigVersion     = "CONFIG_VERSION"
	ErrCodeConfigCompletion  = "CONFIG_COMPLETION"
	ErrCodeConfigStepShape   = "CONFIG_STEP_SHAPE"
	ErrCodeConfigStepName    = "CONFIG_STEP_NAME"
	ErrCodeConfigUnsupported = "CONFIG_UNSUPPORTED_BUILTIN"
	ErrCodeConfigOnFail      = "CONFIG_ON_FAIL"
	ErrCodeConfigGitCommit   = "CONFIG_GIT_COMMIT"
)

type ParseError struct {
	code  string
	msg   string
	cause error
}

func (e *ParseError) Error() string {
	if e.cause == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %v", e.msg, e.cause)
}

func (e *ParseError) Unwrap() error {
	return e.cause
}

func (e *ParseError) Code() string {
	return e.code
}

func (e *ParseError) ExitCode() int {
	return ExitCodeValidation
}

type ValidationError struct {
	code string
	msg  string
}

func (e *ValidationError) Error() string {
	return e.msg
}

func (e *ValidationError) Code() string {
	return e.code
}

func (e *ValidationError) ExitCode() int {
	return ExitCodeValidation
}

func newParseError(msg string, cause error) error {
	return &ParseError{
		code:  ErrCodeConfigParse,
		msg:   msg,
		cause: cause,
	}
}

func newValidationError(code, msg string) error {
	return &ValidationError{
		code: code,
		msg:  msg,
	}
}

func Validate(cfg *Config) error {
	if cfg == nil {
		return newValidationError(ErrCodeConfigParse, "config must not be nil")
	}

	if cfg.Version != SupportedVersion {
		return newValidationError(
			ErrCodeConfigVersion,
			fmt.Sprintf("version must be %q", SupportedVersion),
		)
	}
	cfg.Agent.RunCommand = strings.TrimSpace(cfg.Agent.RunCommand)
	cfg.Agent.ReviewCommand = strings.TrimSpace(cfg.Agent.ReviewCommand)
	cfg.Agent.JudgeCommand = strings.TrimSpace(cfg.Agent.JudgeCommand)
	if cfg.Agent.RunCommand == "" {
		return newValidationError(ErrCodeConfigParse, "agent.run_command is required")
	}

	applyDefaults(cfg)
	if err := validateCompletion(cfg); err != nil {
		return err
	}

	if cfg.Git.Commit != "split" && cfg.Git.Commit != "together" {
		return newValidationError(
			ErrCodeConfigGitCommit,
			fmt.Sprintf("git.commit must be split or together: %q", cfg.Git.Commit),
		)
	}

	if err := validatePhase("pre", &cfg.Phases.Pre); err != nil {
		return err
	}
	if err := validatePhase("post", &cfg.Phases.Post); err != nil {
		return err
	}

	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.Agent.MaxIterations == 0 {
		cfg.Agent.MaxIterations = DefaultAgentMaxIterations
	}
	if cfg.Agent.SleepSeconds == 0 {
		cfg.Agent.SleepSeconds = DefaultAgentSleepSeconds
	}

	cfg.Completion.Run.Strategy = strings.TrimSpace(cfg.Completion.Run.Strategy)
	cfg.Completion.Run.Signal = strings.TrimSpace(cfg.Completion.Run.Signal)
	cfg.Completion.Review.Strategy = strings.TrimSpace(cfg.Completion.Review.Strategy)
	cfg.Completion.Review.Signal = strings.TrimSpace(cfg.Completion.Review.Signal)

	if cfg.Completion.Run.Strategy == "" {
		cfg.Completion.Run.Strategy = DefaultRunCompletionStrategy
	}
	if cfg.Completion.Run.Signal == "" {
		cfg.Completion.Run.Signal = DefaultCompletionSignal
	}
	if cfg.Completion.Run.TailLines == 0 {
		cfg.Completion.Run.TailLines = DefaultRunCompletionTailLines
	}
	if cfg.Completion.Review.Strategy == "" {
		cfg.Completion.Review.Strategy = DefaultReviewCompletionStrategy
	}
	if cfg.Completion.Review.Signal == "" {
		cfg.Completion.Review.Signal = DefaultCompletionSignal
	}
	if cfg.Completion.Review.ReviewConvergence == (ReviewConvergenceMode{}) {
		cfg.Completion.Review.ReviewConvergence = ReviewConvergenceMode{
			MinReviews:   DefaultReviewMinReviews,
			MaxReviews:   DefaultReviewMaxReviews,
			JudgeEvery:   DefaultReviewJudgeEvery,
			StableRounds: DefaultReviewStableRounds,
		}
	}

	if strings.TrimSpace(cfg.Git.Commit) == "" {
		cfg.Git.Commit = DefaultGitCommitMode
	}
	if strings.TrimSpace(cfg.Git.FallbackMessage) == "" {
		cfg.Git.FallbackMessage = DefaultFallbackCommit
	}
}

func validateCompletion(cfg *Config) error {
	if cfg.Completion.Run.Strategy != DefaultRunCompletionStrategy {
		return newValidationError(
			ErrCodeConfigCompletion,
			"completion.run.strategy must be "+DefaultRunCompletionStrategy,
		)
	}
	if cfg.Completion.Review.Strategy != DefaultReviewCompletionStrategy {
		return newValidationError(
			ErrCodeConfigCompletion,
			"completion.review.strategy must be "+DefaultReviewCompletionStrategy,
		)
	}

	review := cfg.Completion.Review.ReviewConvergence
	if review.MinReviews < 1 {
		return newValidationError(
			ErrCodeConfigCompletion,
			"completion.review.review_convergence.min_reviews must be >= 1",
		)
	}
	if review.MaxReviews < review.MinReviews {
		return newValidationError(
			ErrCodeConfigCompletion,
			"completion.review.review_convergence.max_reviews must be >= min_reviews",
		)
	}
	if review.JudgeEvery < 1 {
		return newValidationError(
			ErrCodeConfigCompletion,
			"completion.review.review_convergence.judge_every must be >= 1",
		)
	}
	if review.StableRounds < 1 {
		return newValidationError(
			ErrCodeConfigCompletion,
			"completion.review.review_convergence.stable_rounds must be >= 1",
		)
	}

	return nil
}

func validatePhase(phaseName string, phase *Phase) error {
	seen := make(map[string]struct{}, len(phase.Steps))
	for idx := range phase.Steps {
		step := &phase.Steps[idx]
		name := strings.TrimSpace(step.Name)
		if name == "" {
			return newValidationError(
				ErrCodeConfigStepName,
				fmt.Sprintf("%s step[%d] name is required", phaseName, idx),
			)
		}
		if _, exists := seen[name]; exists {
			return newValidationError(
				ErrCodeConfigStepName,
				fmt.Sprintf("%s step name must be unique: %q", phaseName, name),
			)
		}
		seen[name] = struct{}{}
		step.Name = name

		hasRun := strings.TrimSpace(step.Run) != ""
		hasUses := strings.TrimSpace(step.Uses) != ""
		if hasRun == hasUses {
			return newValidationError(
				ErrCodeConfigStepShape,
				fmt.Sprintf("%s step=%q must set exactly one of run/uses", phaseName, name),
			)
		}

		if hasUses {
			step.Uses = strings.TrimSpace(step.Uses)
			if step.Uses != "auto_commit" {
				return newValidationError(
					ErrCodeConfigUnsupported,
					fmt.Sprintf(
						"%s step=%q uses unsupported builtin %q",
						phaseName,
						name,
						step.Uses,
					),
				)
			}
		}

		if strings.TrimSpace(step.If) == "" {
			step.If = DefaultStepIf
		}
		if _, err := ralphcondition.Parse(step.If); err != nil {
			return newValidationError(
				ErrCodeConfigParse,
				fmt.Sprintf("%s step=%q has invalid if expression: %v", phaseName, name, err),
			)
		}

		onFail := strings.TrimSpace(step.OnFail)
		if onFail == "" {
			onFail = DefaultStepOnFail
		}
		if !isSupportedOnFail(onFail) {
			return newValidationError(
				ErrCodeConfigOnFail,
				fmt.Sprintf("%s step=%q on_fail must be continue or stop_loop", phaseName, name),
			)
		}
		step.OnFail = onFail
	}

	return nil
}

func isSupportedOnFail(value string) bool {
	return value == "continue" || value == "stop_loop"
}
