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

var errUnknownRunCompletionProfileField = errors.New("unknown run completion profile field")

type runCompletionProfileYAML struct {
	Strategy  string `yaml:"strategy"`
	Signal    string `yaml:"signal"`
	TailLines *int   `yaml:"tail_lines"` //nolint:tagliatelle // External schema key uses snake_case.
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
