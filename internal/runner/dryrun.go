package ralphrunner

import (
	"fmt"
	"io"
	"strings"

	ralphconfig "github.com/shuymn/ralph/internal/config"
)

type runPlan struct {
	Agent      ralphconfig.Agent
	Git        ralphconfig.Git
	Completion ralphconfig.Completion
	PreSteps   []ralphconfig.Step
	PostSteps  []ralphconfig.Step
	Paths      fixedPaths
}

func buildRunPlan(cfg ralphconfig.Config, workingDir string) runPlan {
	return runPlan{
		Agent:      cfg.Agent,
		Git:        cfg.Git,
		Completion: cfg.Completion,
		PreSteps:   cfg.Phases.Pre.Steps,
		PostSteps:  cfg.Phases.Post.Steps,
		Paths:      resolvePaths(workingDir),
	}
}

// DryRun renders execution metadata without invoking subprocesses.
func DryRun(cfg ralphconfig.Config, opts Options) int {
	opts = normalizeOptions(opts)
	plan := buildRunPlan(cfg, opts.WorkingDir)

	if _, err := loadPRD(plan.Paths.PRD); err != nil {
		logRuntimeError(opts.Stderr, err)
		return ExitCodeRuntime
	}

	renderDryRunPlan(opts.Stdout, plan)
	return 0
}

func renderDryRunPlan(stdout io.Writer, plan runPlan) {
	_, _ = fmt.Fprintln(stdout, "[ralph] dry-run execution plan")
	_, _ = fmt.Fprintf(stdout, "agent.command: %s\n", plan.Agent.Command)
	_, _ = fmt.Fprintf(stdout, "agent.max_iterations: %d\n", plan.Agent.MaxIterations)
	_, _ = fmt.Fprintf(stdout, "agent.sleep_seconds: %d\n", plan.Agent.SleepSeconds)
	renderDryRunSteps(stdout, "pre", plan.PreSteps)
	renderDryRunSteps(stdout, "post", plan.PostSteps)
	_, _ = fmt.Fprintf(stdout, "git.commit: %s\n", plan.Git.Commit)
	_, _ = fmt.Fprintf(stdout, "git.fallback_message: %s\n", plan.Git.FallbackMessage)
	_, _ = fmt.Fprintf(stdout, "git.fallback_no_gpg_sign: %t\n", plan.Git.FallbackNoGPGSign)
	_, _ = fmt.Fprintf(stdout, "completion.strategy: %s\n", plan.Completion.Strategy)
	_, _ = fmt.Fprintf(stdout, "completion.signal: %s\n", plan.Completion.Signal)
	_, _ = fmt.Fprintf(stdout, "completion.tail_lines: %d\n", plan.Completion.TailLines)
}

func renderDryRunSteps(stdout io.Writer, phase string, steps []ralphconfig.Step) {
	_, _ = fmt.Fprintf(stdout, "phases.%s:\n", phase)
	if len(steps) == 0 {
		_, _ = fmt.Fprintln(stdout, "  (none)")
		return
	}

	for _, step := range steps {
		_, _ = fmt.Fprintf(stdout, "  - name: %s\n", step.Name)
		if runCommand := strings.TrimSpace(step.Run); runCommand != "" {
			_, _ = fmt.Fprintf(stdout, "    run: %s\n", runCommand)
		} else {
			_, _ = fmt.Fprintf(stdout, "    uses: %s\n", strings.TrimSpace(step.Uses))
		}
		_, _ = fmt.Fprintf(stdout, "    if: %s\n", step.If)
		_, _ = fmt.Fprintf(stdout, "    on_fail: %s\n", step.OnFail)
	}
}
