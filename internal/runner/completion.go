package runner

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/shuymn/ralph/internal/config"
	"github.com/shuymn/ralph/internal/prd"
)

type completionResult struct {
	AllPassed    bool
	SignalMatch  bool
	IsCompletion bool
}

func evaluateCompletion(
	prdPath string,
	cfg config.CompletionConfig,
	mainOutputPath string,
) (completionResult, error) {
	doc, err := prd.Load(prdPath)
	if err != nil {
		return completionResult{}, fmt.Errorf("load prd for completion: %w", err)
	}

	allPassed := true
	for _, story := range doc.Stories {
		if !story.Passes {
			allPassed = false
			break
		}
	}

	output, err := os.ReadFile(mainOutputPath)
	if err != nil {
		return completionResult{}, fmt.Errorf("read main output: %w", err)
	}

	tail := tailLines(output, cfg.TailLines)
	signalMatch := slices.Contains(tail, cfg.Signal)

	return completionResult{
		AllPassed:    allPassed,
		SignalMatch:  signalMatch,
		IsCompletion: allPassed && signalMatch,
	}, nil
}

func tailLines(content []byte, count int) []string {
	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if count <= 0 || count >= len(lines) {
		return lines
	}

	return lines[len(lines)-count:]
}
