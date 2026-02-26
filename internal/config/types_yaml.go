package ralphconfig

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

var runCompletionProfileKnownFields = map[string]struct{}{
	"strategy":   {},
	"signal":     {},
	"tail_lines": {},
}

var reviewConvergenceModeKnownFields = map[string]struct{}{
	"min_reviews":   {},
	"max_reviews":   {},
	"judge_every":   {},
	"stable_rounds": {},
}

var (
	errUnknownRunCompletionProfileField  = errors.New("unknown run completion profile field")
	errUnknownReviewConvergenceModeField = errors.New("unknown review convergence mode field")
)

type runCompletionProfileYAML struct {
	Strategy  string `yaml:"strategy"`
	Signal    string `yaml:"signal"`
	TailLines *int   `yaml:"tail_lines"` //nolint:tagliatelle // External schema key uses snake_case.
}

type reviewConvergenceModeYAML struct {
	MinReviews   *int `yaml:"min_reviews"`   //nolint:tagliatelle // External schema key uses snake_case.
	MaxReviews   *int `yaml:"max_reviews"`   //nolint:tagliatelle // External schema key uses snake_case.
	JudgeEvery   *int `yaml:"judge_every"`   //nolint:tagliatelle // External schema key uses snake_case.
	StableRounds *int `yaml:"stable_rounds"` //nolint:tagliatelle // External schema key uses snake_case.
}

func (profile *RunCompletionProfile) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode {
		for idx := 0; idx+1 < len(node.Content); idx += 2 {
			key := node.Content[idx].Value
			if _, ok := runCompletionProfileKnownFields[key]; !ok {
				return fmt.Errorf("%w: %q", errUnknownRunCompletionProfileField, key)
			}
		}
	}

	var decoded runCompletionProfileYAML
	if err := node.Decode(&decoded); err != nil {
		return fmt.Errorf("decode run completion profile: %w", err)
	}

	profile.Strategy = decoded.Strategy
	profile.Signal = decoded.Signal
	profile.tailLinesExplicit = decoded.TailLines != nil
	if decoded.TailLines != nil {
		profile.TailLines = *decoded.TailLines
		return nil
	}

	profile.TailLines = 0
	return nil
}

func (mode *ReviewConvergenceMode) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode {
		for idx := 0; idx+1 < len(node.Content); idx += 2 {
			key := node.Content[idx].Value
			if _, ok := reviewConvergenceModeKnownFields[key]; !ok {
				return fmt.Errorf("%w: %q", errUnknownReviewConvergenceModeField, key)
			}
		}
	}

	var decoded reviewConvergenceModeYAML
	if err := node.Decode(&decoded); err != nil {
		return fmt.Errorf("decode review convergence mode: %w", err)
	}

	mode.minReviewsExplicit = decoded.MinReviews != nil
	if decoded.MinReviews != nil {
		mode.MinReviews = *decoded.MinReviews
	} else {
		mode.MinReviews = 0
	}

	mode.maxReviewsExplicit = decoded.MaxReviews != nil
	if decoded.MaxReviews != nil {
		mode.MaxReviews = *decoded.MaxReviews
	} else {
		mode.MaxReviews = 0
	}

	mode.judgeEveryExplicit = decoded.JudgeEvery != nil
	if decoded.JudgeEvery != nil {
		mode.JudgeEvery = *decoded.JudgeEvery
	} else {
		mode.JudgeEvery = 0
	}

	mode.stableRoundsExplicit = decoded.StableRounds != nil
	if decoded.StableRounds != nil {
		mode.StableRounds = *decoded.StableRounds
	} else {
		mode.StableRounds = 0
	}

	return nil
}
