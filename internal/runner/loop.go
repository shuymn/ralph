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
	promptFileName        = "prompt.md"
	prdFileName           = "prd.json"
	commitMessageFileName = ".commit-msg"
)

var errUnsupportedUsesStep = errors.New("unsupported uses step")

type Options struct {
	WorkingDir string
	Stdout     io.Writer
	Stderr     io.Writer
	TempDir    string
	Sleep      func(time.Duration)
}

type fixedPaths struct {
	Prompt        string
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

func Run(ctx context.Context, cfg ralphconfig.Config, opts Options) int {
	if ctx == nil {
		ctx = context.Background()
	}
	opts = normalizeOptions(opts)
	paths := resolvePaths(opts.WorkingDir)
	autoCommitBase := ralphgit.Options{
		WorkingDir:        opts.WorkingDir,
		Mode:              cfg.Git.Commit,
		FallbackMessage:   cfg.Git.FallbackMessage,
		PRDPath:           paths.PRD,
		CommitMessagePath: paths.CommitMessage,
	}

	tracker := newTmpTracker()
	defer tracker.cleanupAll()

	if _, err := loadPRD(paths.PRD); err != nil {
		logRuntimeError(opts.Stderr, err)
		return ExitCodeRuntime
	}

	for range cfg.Agent.MaxIterations {
		if ctx.Err() != nil {
			logRuntimeError(opts.Stderr, fmt.Errorf("run loop canceled: %w", ctx.Err()))
			return ExitCodeRuntime
		}

		beforePRD, err := loadPRD(paths.PRD)
		if err != nil {
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
		autoCommitOpts := autoCommitBase
		autoCommitOpts.BeforePRD = beforePRD

		changedFunc := ralphcondition.NewGitChangedFunc(opts.WorkingDir)

		preResult, err := runPhase(
			ctx,
			cfg.Phases.Pre.Steps,
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

		mainResult, err := runMainStep(ctx, cfg.Agent.Command, paths.Prompt, opts, tracker)
		if err != nil {
			tracker.cleanup(mainResult.OutputPath)
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}

		postResult, err := runPhase(
			ctx,
			cfg.Phases.Post.Steps,
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

		completionCode, err := completeIteration(mainResult, paths.PRD, cfg.Completion, tracker)
		if err != nil {
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
		if completionCode != noExitCode {
			return completionCode
		}

		sleepDuration := time.Duration(cfg.Agent.SleepSeconds) * time.Second
		if err := sleep(ctx, sleepDuration, opts.Sleep); err != nil {
			logRuntimeError(opts.Stderr, err)
			return ExitCodeRuntime
		}
	}

	return ExitCodeMaxIterations
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
		Prompt:        filepath.Join(base, promptFileName),
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
	completionCfg ralphconfig.Completion,
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
