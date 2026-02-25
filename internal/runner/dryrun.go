package ralphrunner

import (
	"fmt"
	"io"
	"strings"

	ralphconfig "github.com/shuymn/ralph/internal/config"
)

type runPlan struct {
	Agent     ralphconfig.Agent
	Git       ralphconfig.Git
	PreSteps  []ralphconfig.Step
	PostSteps []ralphconfig.Step
	Paths     fixedPaths
}

func buildRunPlan(cfg ralphconfig.Config, workingDir string) runPlan {
	return runPlan{
		Agent:     cfg.Agent,
		Git:       cfg.Git,
		PreSteps:  cfg.Phases.Pre.Steps,
		PostSteps: cfg.Phases.Post.Steps,
		Paths:     resolvePaths(workingDir),
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

	mode := normalizeMode(opts.Mode)
	switch mode {
	case ModeRun:
		runModePlan, err := resolveModePlan(cfg, plan.Paths, ModeRun, RoleRun)
		if err != nil {
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
		renderRunDryRunPlan(opts.Stdout, plan, runModePlan)
	case ModeReview:
		reviewModePlan, err := resolveModePlan(cfg, plan.Paths, ModeReview, RoleReview)
		if err != nil {
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
		judgeModePlan, err := resolveModePlan(cfg, plan.Paths, ModeReview, RoleJudge)
		if err != nil {
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
		renderReviewDryRunPlan(
			opts.Stdout,
			plan,
			reviewModePlan,
			judgeModePlan,
			resolveReviewDryRunProfile(cfg.Completion.Review),
		)
	default:
		logRuntimeError(opts.Stderr, fmt.Errorf("%w: %q", errUnsupportedMode, mode))
		return ExitCodeRuntime
	}

	return 0
}

func renderRunDryRunPlan(stdout io.Writer, plan runPlan, modePlan modePlan) {
	_, _ = fmt.Fprintln(stdout, "[ralph] dry-run execution plan")
	_, _ = fmt.Fprintf(stdout, "mode: %s\n", modePlan.Mode)
	_, _ = fmt.Fprintf(stdout, "agent.run_command: %s\n", modePlan.Command)
	_, _ = fmt.Fprintf(stdout, "agent.max_iterations: %d\n", plan.Agent.MaxIterations)
	_, _ = fmt.Fprintf(stdout, "agent.sleep_seconds: %d\n", plan.Agent.SleepSeconds)
	_, _ = fmt.Fprintf(stdout, "prompt.run_path: %s\n", modePlan.PromptPath)
	renderDryRunPlanCommon(stdout, plan)
	_, _ = fmt.Fprintf(stdout, "completion.run.strategy: %s\n", modePlan.Completion.Strategy)
	_, _ = fmt.Fprintf(stdout, "completion.run.signal: %s\n", modePlan.Completion.Signal)
	_, _ = fmt.Fprintf(stdout, "completion.run.tail_lines: %d\n", modePlan.Completion.TailLines)
}

func renderReviewDryRunPlan(
	stdout io.Writer,
	plan runPlan,
	reviewModePlan modePlan,
	judgeModePlan modePlan,
	profile reviewDryRunProfile,
) {
	_, _ = fmt.Fprintln(stdout, "[ralph] dry-run execution plan")
	_, _ = fmt.Fprintf(stdout, "mode: %s\n", reviewModePlan.Mode)
	_, _ = fmt.Fprintf(stdout, "agent.run_command: %s\n", resolveRunCommand(plan.Agent))
	_, _ = fmt.Fprintf(stdout, "agent.review_command.resolved: %s\n", reviewModePlan.Command)
	_, _ = fmt.Fprintf(
		stdout,
		"agent.review_command.source: %s\n",
		resolveRoleCommandSource(plan.Agent.ReviewCommand, "review_command"),
	)
	_, _ = fmt.Fprintf(stdout, "agent.judge_command.resolved: %s\n", judgeModePlan.Command)
	_, _ = fmt.Fprintf(
		stdout,
		"agent.judge_command.source: %s\n",
		resolveRoleCommandSource(plan.Agent.JudgeCommand, "judge_command"),
	)
	_, _ = fmt.Fprintf(stdout, "agent.max_iterations: %d\n", plan.Agent.MaxIterations)
	_, _ = fmt.Fprintf(stdout, "agent.sleep_seconds: %d\n", plan.Agent.SleepSeconds)
	_, _ = fmt.Fprintf(stdout, "prompt.review_path: %s\n", reviewModePlan.PromptPath)
	_, _ = fmt.Fprintf(stdout, "prompt.judge_path: %s\n", judgeModePlan.PromptPath)
	renderDryRunPlanCommon(stdout, plan)
	_, _ = fmt.Fprintf(stdout, "completion.review.strategy: %s\n", profile.Strategy)
	_, _ = fmt.Fprintf(stdout, "completion.review.signal: %s\n", profile.Signal)
	_, _ = fmt.Fprintf(
		stdout,
		"completion.review.review_convergence.min_reviews: %d\n",
		profile.MinReviews,
	)
	_, _ = fmt.Fprintf(
		stdout,
		"completion.review.review_convergence.max_reviews: %d\n",
		profile.MaxReviews,
	)
	_, _ = fmt.Fprintf(
		stdout,
		"completion.review.review_convergence.judge_every: %d\n",
		profile.JudgeEvery,
	)
	_, _ = fmt.Fprintf(
		stdout,
		"completion.review.review_convergence.stable_rounds: %d\n",
		profile.StableRounds,
	)
	_, _ = fmt.Fprintln(
		stdout,
		"judge_json_contract.required_keys: signal,new_findings,new_finding_keys",
	)
	_, _ = fmt.Fprintln(
		stdout,
		"judge_json_contract.stable_condition: signal == completion.review.signal && new_findings == 0",
	)
}

func renderDryRunPlanCommon(stdout io.Writer, plan runPlan) {
	renderDryRunSteps(stdout, "pre", plan.PreSteps)
	renderDryRunSteps(stdout, "post", plan.PostSteps)
	_, _ = fmt.Fprintf(stdout, "git.commit: %s\n", plan.Git.Commit)
	_, _ = fmt.Fprintf(stdout, "git.fallback_message: %s\n", plan.Git.FallbackMessage)
	_, _ = fmt.Fprintf(stdout, "git.fallback_no_gpg_sign: %t\n", plan.Git.FallbackNoGPGSign)
}

func resolveRoleCommandSource(rawCommand, source string) string {
	if strings.TrimSpace(rawCommand) != "" {
		return source
	}
	return "fallback(run_command)"
}

type reviewDryRunProfile struct {
	Strategy     string
	Signal       string
	MinReviews   int
	MaxReviews   int
	JudgeEvery   int
	StableRounds int
}

func resolveReviewDryRunProfile(profile ralphconfig.ReviewCompletionProfile) reviewDryRunProfile {
	strategy := strings.TrimSpace(profile.Strategy)
	if strategy == "" {
		strategy = ralphconfig.DefaultReviewCompletionStrategy
	}

	signal := strings.TrimSpace(profile.Signal)
	if signal == "" {
		signal = ralphconfig.DefaultCompletionSignal
	}

	minReviews := profile.ReviewConvergence.MinReviews
	if minReviews < 1 {
		minReviews = ralphconfig.DefaultReviewMinReviews
	}
	maxReviews := profile.ReviewConvergence.MaxReviews
	if maxReviews < minReviews {
		maxReviews = max(minReviews, ralphconfig.DefaultReviewMaxReviews)
	}
	judgeEvery := profile.ReviewConvergence.JudgeEvery
	if judgeEvery < 1 {
		judgeEvery = ralphconfig.DefaultReviewJudgeEvery
	}
	stableRounds := profile.ReviewConvergence.StableRounds
	if stableRounds < 1 {
		stableRounds = ralphconfig.DefaultReviewStableRounds
	}

	return reviewDryRunProfile{
		Strategy:     strategy,
		Signal:       signal,
		MinReviews:   minReviews,
		MaxReviews:   maxReviews,
		JudgeEvery:   judgeEvery,
		StableRounds: stableRounds,
	}
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
