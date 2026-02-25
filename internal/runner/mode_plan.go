package ralphrunner

import (
	"errors"
	"fmt"
	"os"
	"strings"

	ralphconfig "github.com/shuymn/ralph/internal/config"
)

var (
	errAgentRunCommandRequired = errors.New("agent.run_command is required")
	errUnsupportedMode         = errors.New("unsupported mode")
	errPromptFileEmpty         = errors.New("prompt file must not be empty")
)

type modePlan struct {
	Mode          Mode
	Role          Role
	Command       string
	PromptPath    string
	RunCompletion ralphconfig.RunCompletionProfile
}

func resolveModePlan(
	cfg ralphconfig.Config,
	paths fixedPaths,
	mode Mode,
	role Role,
) (modePlan, error) {
	resolvedMode := normalizeMode(mode)
	runCommand := resolveRunCommand(cfg.Agent)
	if runCommand == "" {
		return modePlan{}, errAgentRunCommandRequired
	}

	switch resolvedMode {
	case ModeRun:
		if err := requirePrompt(paths.PromptRun); err != nil {
			return modePlan{}, err
		}
		return modePlan{
			Mode:          ModeRun,
			Role:          RoleRun,
			Command:       runCommand,
			PromptPath:    paths.PromptRun,
			RunCompletion: resolveRunCompletion(cfg.Completion.Run),
		}, nil
	case ModeReview:
		if err := requirePrompt(paths.PromptReview); err != nil {
			return modePlan{}, err
		}
		if err := requirePrompt(paths.PromptJudge); err != nil {
			return modePlan{}, err
		}

		resolvedRole := normalizeReviewRole(role)
		return modePlan{
			Mode:          ModeReview,
			Role:          resolvedRole,
			Command:       resolveRoleCommand(cfg.Agent, resolvedRole, runCommand),
			PromptPath:    resolveRolePromptPath(paths, resolvedRole),
			RunCompletion: resolveRunCompletion(cfg.Completion.Run),
		}, nil
	default:
		return modePlan{}, fmt.Errorf("%w: %q", errUnsupportedMode, resolvedMode)
	}
}

func resolveRunCommand(agent ralphconfig.Agent) string {
	return strings.TrimSpace(agent.RunCommand)
}

func resolveRoleCommand(agent ralphconfig.Agent, role Role, runCommand string) string {
	switch role {
	case RoleRun:
		return runCommand
	case RoleReview:
		if command := strings.TrimSpace(agent.ReviewCommand); command != "" {
			return command
		}
	case RoleJudge:
		if command := strings.TrimSpace(agent.JudgeCommand); command != "" {
			return command
		}
	}

	return runCommand
}

func resolveRolePromptPath(paths fixedPaths, role Role) string {
	switch role {
	case RoleRun:
		return paths.PromptRun
	case RoleReview:
		return paths.PromptReview
	case RoleJudge:
		return paths.PromptJudge
	}

	return paths.PromptReview
}

func normalizeMode(mode Mode) Mode {
	if strings.TrimSpace(string(mode)) == "" {
		return ModeRun
	}
	return mode
}

func normalizeReviewRole(role Role) Role {
	switch role {
	case RoleRun:
		return RoleReview
	case RoleJudge:
		return RoleJudge
	case RoleReview:
		return RoleReview
	default:
		return RoleReview
	}
}

func resolveRunCompletion(
	profile ralphconfig.RunCompletionProfile,
) ralphconfig.RunCompletionProfile {
	resolved := profile

	strategy := strings.TrimSpace(resolved.Strategy)
	if strategy == "" {
		strategy = ralphconfig.DefaultRunCompletionStrategy
	}

	signal := strings.TrimSpace(resolved.Signal)
	if signal == "" {
		signal = ralphconfig.DefaultCompletionSignal
	}

	tailLines := resolved.TailLines
	if tailLines <= 0 {
		tailLines = ralphconfig.DefaultRunCompletionTailLines
	}

	resolved.Strategy = strategy
	resolved.Signal = signal
	resolved.TailLines = tailLines

	return resolved
}

func requirePrompt(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read prompt file %q: %w", path, err)
	}
	if strings.TrimSpace(string(content)) == "" {
		return fmt.Errorf("%w: %q", errPromptFileEmpty, path)
	}

	return nil
}
