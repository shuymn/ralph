package config

import "fmt"

func Validate(cfg *Config) error {
	if cfg == nil {
		return newValidationError("config is required")
	}

	if cfg.Version != SupportedVersion {
		return newValidationError(`version must be "1"`)
	}

	if cfg.Agent.Command == "" {
		return newValidationError("agent.command is required")
	}

	switch cfg.Git.Commit {
	case DefaultCommitMode, CommitModeTogether:
	default:
		return newValidationError(
			fmt.Sprintf(
				"invalid git.commit %q: allowed values are %q or %q",
				cfg.Git.Commit,
				DefaultCommitMode,
				CommitModeTogether,
			),
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

func validatePhase(phaseName string, phase *PhaseConfig) error {
	if phase == nil {
		return nil
	}

	seenNames := make(map[string]struct{}, len(phase.Steps))

	for i := range phase.Steps {
		step := &phase.Steps[i]
		stepPath := formatStepPath(phaseName, i)

		if step.Name == "" {
			return newValidationError(stepPath + ".name is required")
		}

		if _, exists := seenNames[step.Name]; exists {
			return newValidationError(
				fmt.Sprintf("duplicate step name %q in phase %q", step.Name, phaseName),
			)
		}
		seenNames[step.Name] = struct{}{}

		hasRun := step.Run != ""
		hasUses := step.Uses != ""
		if hasRun == hasUses {
			return newValidationError(stepPath + " must define exactly one of run or uses")
		}

		if hasUses && step.Uses != BuiltinAutoCommit {
			return newValidationError(
				fmt.Sprintf("unsupported builtin %q in %s", step.Uses, stepPath),
			)
		}

		switch step.OnFail {
		case DefaultOnFail, OnFailContinue:
		default:
			return newValidationError(
				fmt.Sprintf("invalid on_fail %q in %s", step.OnFail, stepPath),
			)
		}
	}

	return nil
}
