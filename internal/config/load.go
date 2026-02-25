package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, newParseError("read config file", err)
	}

	var raw any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return Config{}, newParseError("decode yaml", err)
	}

	root, err := asMap(raw, "config")
	if err != nil {
		return Config{}, err
	}

	cfg, err := parseConfig(root)
	if err != nil {
		return Config{}, err
	}

	applyDefaults(&cfg)

	if err := Validate(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func parseConfig(root map[string]any) (Config, error) {
	cfg := Config{}

	if version, ok, err := getString(root, "version", "version"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.Version = version
	}

	if agentRaw, ok := root["agent"]; ok {
		agentMap, err := asMap(agentRaw, "agent")
		if err != nil {
			return Config{}, err
		}
		agent, err := parseAgent(agentMap)
		if err != nil {
			return Config{}, err
		}
		cfg.Agent = agent
	}

	if completionRaw, ok := root["completion"]; ok {
		completionMap, err := asMap(completionRaw, "completion")
		if err != nil {
			return Config{}, err
		}
		completion, err := parseCompletion(completionMap)
		if err != nil {
			return Config{}, err
		}
		cfg.Completion = completion
	}

	if gitRaw, ok := root["git"]; ok {
		gitMap, err := asMap(gitRaw, "git")
		if err != nil {
			return Config{}, err
		}
		git, err := parseGit(gitMap)
		if err != nil {
			return Config{}, err
		}
		cfg.Git = git
	}

	if phasesRaw, ok := root["phases"]; ok {
		phasesMap, err := asMap(phasesRaw, "phases")
		if err != nil {
			return Config{}, err
		}
		phases, err := parsePhases(phasesMap)
		if err != nil {
			return Config{}, err
		}
		cfg.Phases = phases
	}

	return cfg, nil
}

func parseAgent(raw map[string]any) (AgentConfig, error) {
	agent := AgentConfig{}

	if command, ok, err := getString(raw, "command", "agent.command"); err != nil {
		return AgentConfig{}, err
	} else if ok {
		agent.Command = command
	}

	if maxIterations, ok, err := getInt(raw, "max_iterations", "agent.max_iterations"); err != nil {
		return AgentConfig{}, err
	} else if ok {
		agent.MaxIterations = maxIterations
	}

	if sleepSeconds, ok, err := getInt(raw, "sleep_seconds", "agent.sleep_seconds"); err != nil {
		return AgentConfig{}, err
	} else if ok {
		agent.SleepSeconds = sleepSeconds
	}

	return agent, nil
}

func parseCompletion(raw map[string]any) (CompletionConfig, error) {
	completion := CompletionConfig{}

	if strategy, ok, err := getString(raw, "strategy", "completion.strategy"); err != nil {
		return CompletionConfig{}, err
	} else if ok {
		completion.Strategy = strategy
	}

	if signal, ok, err := getString(raw, "signal", "completion.signal"); err != nil {
		return CompletionConfig{}, err
	} else if ok {
		completion.Signal = signal
	}

	if tailLines, ok, err := getInt(raw, "tail_lines", "completion.tail_lines"); err != nil {
		return CompletionConfig{}, err
	} else if ok {
		completion.TailLines = tailLines
	}

	return completion, nil
}

func parseGit(raw map[string]any) (GitConfig, error) {
	git := GitConfig{}

	if commit, ok, err := getString(raw, "commit", "git.commit"); err != nil {
		return GitConfig{}, err
	} else if ok {
		git.Commit = commit
	}

	if fallbackMessage, ok, err := getString(
		raw,
		"fallback_message",
		"git.fallback_message",
	); err != nil {
		return GitConfig{}, err
	} else if ok {
		git.FallbackMessage = fallbackMessage
	}

	return git, nil
}

func parsePhases(raw map[string]any) (PhasesConfig, error) {
	phases := PhasesConfig{}

	if preRaw, ok := raw["pre"]; ok {
		preMap, err := asMap(preRaw, "phases.pre")
		if err != nil {
			return PhasesConfig{}, err
		}
		pre, err := parsePhase(preMap, "phases.pre")
		if err != nil {
			return PhasesConfig{}, err
		}
		phases.Pre = pre
	}

	if postRaw, ok := raw["post"]; ok {
		postMap, err := asMap(postRaw, "phases.post")
		if err != nil {
			return PhasesConfig{}, err
		}
		post, err := parsePhase(postMap, "phases.post")
		if err != nil {
			return PhasesConfig{}, err
		}
		phases.Post = post
	}

	return phases, nil
}

func parsePhase(raw map[string]any, path string) (PhaseConfig, error) {
	phase := PhaseConfig{}

	stepsRaw, ok := raw["steps"]
	if !ok {
		return phase, nil
	}

	stepsList, err := asList(stepsRaw, path+".steps")
	if err != nil {
		return PhaseConfig{}, err
	}

	phase.Steps = make([]StepConfig, 0, len(stepsList))
	for i, rawStep := range stepsList {
		stepMap, err := asMap(rawStep, formatStepPath(path, i))
		if err != nil {
			return PhaseConfig{}, err
		}
		step, err := parseStep(stepMap, path, i)
		if err != nil {
			return PhaseConfig{}, err
		}
		phase.Steps = append(phase.Steps, step)
	}

	return phase, nil
}

func parseStep(raw map[string]any, phasePath string, index int) (StepConfig, error) {
	stepPath := formatStepPath(phasePath, index)
	step := StepConfig{}

	if name, ok, err := getString(raw, "name", stepPath+".name"); err != nil {
		return StepConfig{}, err
	} else if ok {
		step.Name = name
	}

	if run, ok, err := getString(raw, "run", stepPath+".run"); err != nil {
		return StepConfig{}, err
	} else if ok {
		step.Run = run
	}

	if uses, ok, err := getString(raw, "uses", stepPath+".uses"); err != nil {
		return StepConfig{}, err
	} else if ok {
		step.Uses = uses
	}

	if ifExpr, ok, err := getString(raw, "if", stepPath+".if"); err != nil {
		return StepConfig{}, err
	} else if ok {
		step.If = ifExpr
	}

	if onFail, ok, err := getString(raw, "on_fail", stepPath+".on_fail"); err != nil {
		return StepConfig{}, err
	} else if ok {
		step.OnFail = onFail
	}

	return step, nil
}

func applyDefaults(cfg *Config) {
	if cfg == nil {
		return
	}

	cfg.Version = strings.TrimSpace(cfg.Version)
	cfg.Agent.Command = strings.TrimSpace(cfg.Agent.Command)

	if cfg.Agent.MaxIterations == 0 {
		cfg.Agent.MaxIterations = DefaultAgentIterations
	}
	if cfg.Agent.SleepSeconds == 0 {
		cfg.Agent.SleepSeconds = DefaultAgentSleep
	}

	cfg.Completion.Strategy = strings.TrimSpace(cfg.Completion.Strategy)
	if cfg.Completion.Strategy == "" {
		cfg.Completion.Strategy = DefaultCompletionMode
	}
	if cfg.Completion.TailLines == 0 {
		cfg.Completion.TailLines = DefaultCompletionTail
	}
	if strings.TrimSpace(cfg.Completion.Signal) == "" {
		cfg.Completion.Signal = DefaultCompletionSig
	}

	cfg.Git.Commit = strings.TrimSpace(cfg.Git.Commit)
	if cfg.Git.Commit == "" {
		cfg.Git.Commit = DefaultCommitMode
	}
	if strings.TrimSpace(cfg.Git.FallbackMessage) == "" {
		cfg.Git.FallbackMessage = DefaultFallbackMessage
	}

	applyStepDefaults(&cfg.Phases.Pre)
	applyStepDefaults(&cfg.Phases.Post)
}

func applyStepDefaults(phase *PhaseConfig) {
	if phase == nil {
		return
	}

	for i := range phase.Steps {
		step := &phase.Steps[i]
		step.Name = strings.TrimSpace(step.Name)
		step.Run = strings.TrimSpace(step.Run)
		step.Uses = strings.TrimSpace(step.Uses)
		step.If = strings.TrimSpace(step.If)
		step.OnFail = strings.TrimSpace(step.OnFail)

		if step.If == "" {
			step.If = DefaultIfExpr
		}
		if step.OnFail == "" {
			step.OnFail = DefaultOnFail
		}
	}
}

func asMap(value any, path string) (map[string]any, error) {
	mapValue, ok := value.(map[string]any)
	if !ok {
		return nil, newParseError(path+" must be a mapping", nil)
	}
	return mapValue, nil
}

func asList(value any, path string) ([]any, error) {
	listValue, ok := value.([]any)
	if !ok {
		return nil, newParseError(path+" must be a sequence", nil)
	}
	return listValue, nil
}

func getString(raw map[string]any, key string, path string) (string, bool, error) {
	value, ok := raw[key]
	if !ok {
		return "", false, nil
	}

	stringValue, ok := value.(string)
	if !ok {
		return "", false, newParseError(path+" must be a string", nil)
	}

	return stringValue, true, nil
}

func getInt(raw map[string]any, key string, path string) (int, bool, error) {
	value, ok := raw[key]
	if !ok {
		return 0, false, nil
	}

	switch typed := value.(type) {
	case int:
		return typed, true, nil
	case int64:
		return int(typed), true, nil
	case float64:
		if typed != float64(int(typed)) {
			return 0, false, newParseError(path+" must be an integer", nil)
		}
		return int(typed), true, nil
	default:
		return 0, false, newParseError(path+" must be an integer", nil)
	}
}

func formatStepPath(phasePath string, index int) string {
	return fmt.Sprintf("%s.steps[%d]", phasePath, index)
}
