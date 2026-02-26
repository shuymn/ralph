package ralphrunner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	ralphcondition "github.com/shuymn/ralph/internal/condition"
	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphgit "github.com/shuymn/ralph/internal/git"
	ralphprd "github.com/shuymn/ralph/internal/prd"
)

const (
	ExitCodeStopLoop           = 20
	ExitCodeCompletionMismatch = 21
	ExitCodeRuntime            = 22
	ExitCodeMaxIterations      = 23
)

const noExitCode = -1

const (
	ralphDirName          = ".ralph"
	promptRunFileName     = "prompt.run.md"
	promptReviewFileName  = "prompt.review.md"
	promptJudgeFileName   = "prompt.judge.md"
	legacyPromptFileName  = "prompt.md"
	prdFileName           = "prd.json"
	commitMessageFileName = ".commit-msg"
)

var (
	errUnsupportedUsesStep      = errors.New("unsupported uses step")
	errJudgeArtifactPathMissing = errors.New("judge artifact path must not be empty")
	errJudgeCommandFailed       = errors.New("judge command failed")
)

type Options struct {
	WorkingDir string
	Stdout     io.Writer
	Stderr     io.Writer
	TempDir    string
	Sleep      func(time.Duration)
	Mode       Mode
	Role       Role
}

type Mode string

const (
	ModeRun    Mode = "run"
	ModeReview Mode = "review"
)

type Role string

const (
	RoleRun    Role = "run"
	RoleReview Role = "review"
	RoleJudge  Role = "judge"
)

type fixedPaths struct {
	Prompt        string
	PromptRun     string
	PromptReview  string
	PromptJudge   string
	PRD           string
	CommitMessage string
}

type phaseState struct {
	success bool
	failure bool
}

type phaseResult struct {
	stopLoop bool
	stepName string
	reason   string
}

type reviewRuntime struct {
	state      reviewState
	config     ralphconfig.ReviewConvergenceMode
	artifacts  reviewArtifacts
	completion reviewCompletion
}

func Run(ctx context.Context, cfg ralphconfig.Config, opts Options) int {
	if ctx == nil {
		ctx = context.Background()
	}
	opts = normalizeOptions(opts)
	plan := buildRunPlan(cfg, opts.WorkingDir)
	paths := plan.Paths
	modePlan, err := resolveModePlan(cfg, paths, opts.Mode, opts.Role)
	if err != nil {
		logRuntimeError(opts.Stderr, err)
		return ExitCodeRuntime
	}
	reviewMode := modePlan.Mode == ModeReview
	var reviewRuntimeState *reviewRuntime
	enableReviewScheduler := reviewMode && strings.TrimSpace(string(opts.Role)) == ""
	if reviewMode {
		runtime := newReviewRuntime(
			paths,
			cfg.Completion.Review,
			time.Now(),
		)
		reviewRuntimeState = &runtime
	}
	autoCommitBase := ralphgit.Options{
		WorkingDir:        opts.WorkingDir,
		Mode:              plan.Git.Commit,
		FallbackMessage:   plan.Git.FallbackMessage,
		FallbackNoGPGSign: plan.Git.FallbackNoGPGSign,
		PRDPath:           paths.PRD,
		CommitMessagePath: paths.CommitMessage,
	}

	tracker := newTmpTracker()
	defer tracker.cleanupAll()

	initialPRD, err := loadPRD(paths.PRD)
	if err != nil {
		logRuntimeError(opts.Stderr, err)
		return ExitCodeRuntime
	}
	if err := ralphgit.EnsureBranch(ctx, ralphgit.EnsureBranchOptions{
		WorkingDir: opts.WorkingDir,
		BranchName: initialPRD.BranchName,
	}); err != nil {
		logRuntimeError(opts.Stderr, fmt.Errorf("ensure branch: %w", err))
		return ExitCodeRuntime
	}

	for range plan.Agent.MaxIterations {
		if ctx.Err() != nil {
			logRuntimeError(opts.Stderr, fmt.Errorf("run loop canceled: %w", ctx.Err()))
			return ExitCodeRuntime
		}

		beforePRD, err := loadPRD(paths.PRD)
		if err != nil {
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
		iterationPlan := modePlan
		if enableReviewScheduler {
			scheduledRole := nextReviewRole(
				reviewRuntimeState.state,
				reviewRuntimeState.config,
			)
			iterationPlan, err = resolveModePlan(
				cfg,
				paths,
				ModeReview,
				scheduledRole,
			)
			if err != nil {
				logRuntimeError(opts.Stderr, err)
				return ExitCodeRuntime
			}
		}

		autoCommitOpts := autoCommitBase
		autoCommitOpts.BeforePRD = beforePRD

		phasesEnabled := iterationPlan.Mode == ModeRun
		changedFunc := ralphcondition.NewGitChangedFunc(opts.WorkingDir)

		if phasesEnabled {
			preResult, err := runPhase(
				ctx,
				plan.PreSteps,
				&phaseState{success: true, failure: false},
				changedFunc,
				autoCommitOpts,
				opts,
			)
			if err != nil {
				logRuntimeError(opts.Stderr, err)
				return ExitCodeRuntime
			}
			if preResult.stopLoop {
				logStopLoop(opts.Stderr, "pre", preResult.stepName, preResult.reason)
				return ExitCodeStopLoop
			}
		}

		inputPrefix := ""
		if reviewRuntimeState != nil && iterationPlan.Role == RoleJudge {
			inputPrefix = buildJudgeInputPrefix(*reviewRuntimeState)
		}
		mainResult, err := runMainStep(ctx, mainStepPlan{
			Role:        iterationPlan.Role,
			Command:     iterationPlan.Command,
			PromptPath:  iterationPlan.PromptPath,
			InputPrefix: inputPrefix,
		}, opts, tracker)
		if err != nil {
			tracker.cleanup(mainResult.OutputPath)
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
		reviewArtifactPath := ""
		if reviewRuntimeState != nil {
			artifactPath, err := reviewRuntimeState.artifacts.nextPath(
				iterationPlan.Role,
				reviewRuntimeState.state,
			)
			if err != nil {
				tracker.cleanup(mainResult.OutputPath)
				logRuntimeError(opts.Stderr, err)
				return ExitCodeRuntime
			}
			if err := reviewRuntimeState.artifacts.persist(
				artifactPath,
				mainResult.OutputPath,
			); err != nil {
				tracker.cleanup(mainResult.OutputPath)
				logRuntimeError(opts.Stderr, err)
				return ExitCodeRuntime
			}
			reviewArtifactPath = artifactPath
			reviewRuntimeState.state.recordRole(iterationPlan.Role)
		}

		if phasesEnabled {
			postResult, err := runPhase(
				ctx,
				plan.PostSteps,
				&phaseState{success: mainResult.Success, failure: !mainResult.Success},
				changedFunc,
				autoCommitOpts,
				opts,
			)
			if err != nil {
				tracker.cleanup(mainResult.OutputPath)
				logRuntimeError(opts.Stderr, err)
				return ExitCodeRuntime
			}
			if postResult.stopLoop {
				tracker.cleanup(mainResult.OutputPath)
				logStopLoop(opts.Stderr, "post", postResult.stepName, postResult.reason)
				return ExitCodeStopLoop
			}
		}

		completionCode, err := completeModeIteration(
			iterationPlan,
			mainResult,
			reviewArtifactPath,
			reviewRuntimeState,
			paths.PRD,
			tracker,
		)
		if err != nil {
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
		if completionCode != noExitCode {
			return completionCode
		}

		sleepDuration := time.Duration(plan.Agent.SleepSeconds) * time.Second
		if err := sleep(ctx, sleepDuration, opts.Sleep); err != nil {
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
	}

	return ExitCodeMaxIterations
}

func newReviewRuntime(
	paths fixedPaths,
	profile ralphconfig.ReviewCompletionProfile,
	now time.Time,
) reviewRuntime {
	state := newReviewState(now)
	return reviewRuntime{
		state:      state,
		config:     profile.ReviewConvergence,
		artifacts:  newReviewArtifacts(paths, state.runID),
		completion: newReviewCompletion(profile),
	}
}

func runPhase(
	ctx context.Context,
	steps []ralphconfig.Step,
	state *phaseState,
	changed ralphcondition.ChangedFunc,
	autoCommitOpts ralphgit.Options,
	opts Options,
) (phaseResult, error) {
	for _, step := range steps {
		if ctx.Err() != nil {
			return phaseResult{}, fmt.Errorf("phase canceled: %w", ctx.Err())
		}

		condition, err := ralphcondition.Eval(
			step.If,
			ralphcondition.NewContext(state.success, state.failure, changed),
		)
		if err != nil {
			return phaseResult{}, fmt.Errorf(
				"evaluate if expression for step %q: %w",
				step.Name,
				err,
			)
		}
		if !condition {
			continue
		}

		err = runStep(ctx, step, autoCommitOpts, opts)
		if err == nil {
			continue
		}

		if step.OnFail == "stop_loop" {
			return phaseResult{stopLoop: true, stepName: step.Name, reason: err.Error()}, nil
		}
		state.success = false
		state.failure = true
	}

	return phaseResult{}, nil
}

func runStep(
	ctx context.Context,
	step ralphconfig.Step,
	autoCommitOpts ralphgit.Options,
	opts Options,
) error {
	runCommand := strings.TrimSpace(step.Run)
	if runCommand != "" {
		return runStepCommand(ctx, runCommand, opts)
	}

	uses := strings.TrimSpace(step.Uses)
	if uses == "auto_commit" {
		if err := ralphgit.AutoCommit(ctx, autoCommitOpts); err != nil {
			return fmt.Errorf("run auto_commit step: %w", err)
		}
		return nil
	}

	return fmt.Errorf("%w: %q", errUnsupportedUsesStep, uses)
}

func resolvePaths(workingDir string) fixedPaths {
	base := filepath.Join(workingDir, ralphDirName)
	return fixedPaths{
		Prompt:        filepath.Join(base, legacyPromptFileName),
		PromptRun:     filepath.Join(base, promptRunFileName),
		PromptReview:  filepath.Join(base, promptReviewFileName),
		PromptJudge:   filepath.Join(base, promptJudgeFileName),
		PRD:           filepath.Join(base, prdFileName),
		CommitMessage: filepath.Join(base, commitMessageFileName),
	}
}

func loadPRD(path string) (ralphprd.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ralphprd.Document{}, fmt.Errorf("read prd: %w", err)
	}

	doc, err := ralphprd.ValidateBytes(content)
	if err != nil {
		return ralphprd.Document{}, fmt.Errorf("validate prd: %w", err)
	}

	return doc, nil
}

func normalizeOptions(opts Options) Options {
	if strings.TrimSpace(opts.WorkingDir) == "" {
		opts.WorkingDir = "."
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if strings.TrimSpace(string(opts.Mode)) == "" {
		opts.Mode = ModeRun
	}

	return opts
}

func sleep(ctx context.Context, duration time.Duration, sleepFn func(time.Duration)) error {
	if duration <= 0 {
		return nil
	}

	if sleepFn != nil {
		sleepFn(duration)
		if ctx.Err() != nil {
			return fmt.Errorf("sleep canceled: %w", ctx.Err())
		}
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("sleep canceled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func completeIteration(
	mainResult mainResult,
	prdPath string,
	completionCfg ralphconfig.RunCompletionProfile,
	tracker *tmpTracker,
) (int, error) {
	defer tracker.cleanup(mainResult.OutputPath)

	if !mainResult.Success {
		return noExitCode, nil
	}

	doc, err := loadPRD(prdPath)
	if err != nil {
		return noExitCode, fmt.Errorf("load prd for completion: %w", err)
	}

	complete, mismatch, err := evaluateCompletion(mainResult.OutputPath, doc, completionCfg)
	if err != nil {
		return noExitCode, fmt.Errorf("evaluate completion: %w", err)
	}

	if complete {
		return 0, nil
	}
	if mismatch {
		return ExitCodeCompletionMismatch, nil
	}

	return noExitCode, nil
}

func completeModeIteration(
	iterationPlan modePlan,
	mainResult mainResult,
	reviewArtifactPath string,
	reviewRuntimeState *reviewRuntime,
	prdPath string,
	tracker *tmpTracker,
) (int, error) {
	if reviewRuntimeState != nil {
		return completeReviewIteration(
			iterationPlan.Role,
			reviewArtifactPath,
			mainResult,
			reviewRuntimeState,
			tracker,
		)
	}

	return completeIteration(
		mainResult,
		prdPath,
		iterationPlan.RunCompletion,
		tracker,
	)
}

func completeReviewIteration(
	role Role,
	artifactPath string,
	mainResult mainResult,
	runtime *reviewRuntime,
	tracker *tmpTracker,
) (int, error) {
	defer tracker.cleanup(mainResult.OutputPath)

	if role != RoleJudge {
		return noExitCode, nil
	}
	if !mainResult.Success {
		return noExitCode, errJudgeCommandFailed
	}
	if artifactPath == "" {
		return noExitCode, errJudgeArtifactPathMissing
	}

	judgeResult, err := parseJudgeContract(artifactPath)
	if err != nil {
		return noExitCode, fmt.Errorf("parse judge contract: %w", err)
	}
	if err := runtime.completion.recordJudge(judgeResult); err != nil {
		return noExitCode, fmt.Errorf("record judge contract: %w", err)
	}
	if runtime.completion.converged(runtime.state.reviewCount) {
		return 0, nil
	}
	if runtime.completion.nonConvergedAtMax(runtime.state.reviewCount) {
		return ExitCodeMaxIterations, nil
	}

	return noExitCode, nil
}

func logStopLoop(stderr io.Writer, phase, step, reason string) {
	_, _ = fmt.Fprintf(
		stderr,
		"[ralph] stop_loop phase=%s step=%s reason=%s\n",
		phase,
		step,
		reason,
	)
}

func logRuntimeError(stderr io.Writer, err error) {
	_, _ = fmt.Fprintf(stderr, "[ralph] runtime_error: %v\n", err)
}
