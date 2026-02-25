package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shuymn/ralph/internal/condition"
	"github.com/shuymn/ralph/internal/config"
	ralphgit "github.com/shuymn/ralph/internal/git"
	"github.com/shuymn/ralph/internal/prd"
)

const (
	ExitCodeSuccess          = 0
	ExitCodeStopLoop         = 20
	ExitCodeProtocolMismatch = 21
	ExitCodeConfigError      = 22
	ExitCodeMaxIterations    = 23
)

const (
	phasePre  = "pre"
	phasePost = "post"

	ralphDirName   = ".ralph"
	configFileName = "config.yml"
	promptFileName = "prompt.md"
	prdFileName    = "prd.json"
	commitMsgFile  = ".commit-msg"
)

var errUnsupportedBuiltin = errors.New("unsupported builtin")

type Options struct {
	WorkDir    string
	ConfigPath string
	PromptPath string
	PRDPath    string
	Stdout     io.Writer
	Stderr     io.Writer
	Signals    <-chan os.Signal
	Sleep      func(time.Duration)
}

type compiledStep struct {
	ConfigStep config.StepConfig
	Condition  condition.Expression
}

type builtinContext struct {
	Git               config.GitConfig
	CommitMessagePath string
	PRDPath           string
	BeforePRD         prd.Document
}

type executionPlan struct {
	Config    config.Config
	PreSteps  []compiledStep
	PostSteps []compiledStep
	PRD       prd.Document
}

func Run(opts Options) int {
	resolved, plan, err := loadExecutionPlan(opts)
	if err != nil {
		logError(resolved.Stderr, err)
		return ExitCodeConfigError
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalReceived := &atomic.Bool{}
	if resolved.Signals != nil {
		go watchSignals(ctx, resolved.Signals, signalReceived, cancel)
	}

	return runLoop(
		ctx,
		resolved,
		plan.Config,
		plan.PreSteps,
		plan.PostSteps,
		signalReceived,
	)
}

func loadExecutionPlan(opts Options) (Options, executionPlan, error) {
	resolved := resolveOptions(opts)

	cfg, err := config.Load(resolved.ConfigPath)
	if err != nil {
		return resolved, executionPlan{}, fmt.Errorf("load config: %w", err)
	}

	compiledPre, err := compilePhase(phasePre, cfg.Phases.Pre.Steps)
	if err != nil {
		return resolved, executionPlan{}, err
	}

	compiledPost, err := compilePhase(phasePost, cfg.Phases.Post.Steps)
	if err != nil {
		return resolved, executionPlan{}, err
	}

	doc, err := prd.Load(resolved.PRDPath)
	if err != nil {
		return resolved, executionPlan{}, fmt.Errorf("load prd: %w", err)
	}

	return resolved, executionPlan{
		Config:    cfg,
		PreSteps:  compiledPre,
		PostSteps: compiledPost,
		PRD:       doc,
	}, nil
}

func runLoop(
	ctx context.Context,
	opts Options,
	cfg config.Config,
	preSteps []compiledStep,
	postSteps []compiledStep,
	signalReceived *atomic.Bool,
) int {
	for iteration := range cfg.Agent.MaxIterations {
		if signalReceived.Load() {
			return ExitCodeStopLoop
		}

		beforePRD, err := prd.Load(opts.PRDPath)
		if err != nil {
			logError(opts.Stderr, err)
			return ExitCodeConfigError
		}

		builtinOpts := builtinContext{
			Git:               cfg.Git,
			CommitMessagePath: filepath.Join(opts.WorkDir, ralphDirName, commitMsgFile),
			PRDPath:           opts.PRDPath,
			BeforePRD:         beforePRD,
		}

		preResult, err := executePhase(
			ctx,
			opts.WorkDir,
			phasePre,
			preSteps,
			true,
			builtinOpts,
			opts.Stdout,
			opts.Stderr,
		)
		if err != nil {
			logError(opts.Stderr, err)
			return ExitCodeConfigError
		}
		if preResult.StopLoop {
			return ExitCodeStopLoop
		}

		mainResult := runMainStep(
			ctx,
			opts.WorkDir,
			opts.PromptPath,
			cfg.Agent.Command,
			opts.Stderr,
		)
		if mainResult.Err != nil {
			logError(opts.Stderr, mainResult.Err)
		}

		postResult, err := executePhase(
			ctx,
			opts.WorkDir,
			phasePost,
			postSteps,
			mainResult.Success,
			builtinOpts,
			opts.Stdout,
			opts.Stderr,
		)
		if err != nil {
			mainResult.Output.Cleanup(opts.Stderr)
			logError(opts.Stderr, err)
			return ExitCodeConfigError
		}
		if postResult.StopLoop {
			mainResult.Output.Cleanup(opts.Stderr)
			return ExitCodeStopLoop
		}

		if mainResult.Success {
			completion, err := evaluateCompletion(
				opts.PRDPath,
				cfg.Completion,
				mainResult.Output.Path,
			)
			if err != nil {
				mainResult.Output.Cleanup(opts.Stderr)
				logError(opts.Stderr, err)
				return ExitCodeConfigError
			}

			if completion.AllPassed && !completion.SignalMatch {
				mainResult.Output.Cleanup(opts.Stderr)
				fmt.Fprintf(
					opts.Stderr,
					"[ralph] completion protocol mismatch: signal %q not found in tail\n",
					cfg.Completion.Signal,
				)
				return ExitCodeProtocolMismatch
			}

			if completion.IsCompletion {
				mainResult.Output.Cleanup(opts.Stderr)
				return ExitCodeSuccess
			}
		}

		mainResult.Output.Cleanup(opts.Stderr)

		if signalReceived.Load() {
			return ExitCodeStopLoop
		}

		if iteration == cfg.Agent.MaxIterations-1 {
			break
		}

		if !waitForNextIteration(ctx, cfg.Agent.SleepSeconds, opts.Sleep) {
			return ExitCodeStopLoop
		}
	}

	return ExitCodeMaxIterations
}

type phaseResult struct {
	StopLoop bool
}

func executePhase(
	ctx context.Context,
	workDir string,
	phaseName string,
	steps []compiledStep,
	phaseSuccess bool,
	builtins builtinContext,
	stdout io.Writer,
	stderr io.Writer,
) (phaseResult, error) {
	stateSuccess := phaseSuccess

	for _, step := range steps {
		shouldRun, err := step.Condition.Evaluate(condition.Context{
			Success: stateSuccess,
			Changed: condition.GitChanged(workDir),
		})
		if err != nil {
			return phaseResult{}, fmt.Errorf(
				"evaluate if for phase=%s step=%s: %w",
				phaseName,
				step.ConfigStep.Name,
				err,
			)
		}
		if !shouldRun {
			continue
		}

		if err := runStep(
			ctx,
			workDir,
			step.ConfigStep,
			builtins,
			stdout,
			stderr,
		); err != nil {
			stateSuccess = false
			if step.ConfigStep.OnFail == config.OnFailContinue {
				continue
			}

			logStopLoop(stderr, phaseName, step.ConfigStep.Name, err)
			return phaseResult{StopLoop: true}, nil
		}
	}

	return phaseResult{}, nil
}

func runStep(
	ctx context.Context,
	workDir string,
	step config.StepConfig,
	builtins builtinContext,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if step.Run != "" {
		return runShellStep(ctx, workDir, step.Run, stdout, stderr)
	}

	return runBuiltinStep(ctx, workDir, step, builtins)
}

func runShellStep(
	ctx context.Context,
	workDir string,
	command string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	cmd := exec.CommandContext(ctx, shellCommand, shellFlag, command)
	cmd.Dir = workDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run step command %q: %w", command, err)
	}

	return nil
}

func runBuiltinStep(
	ctx context.Context,
	workDir string,
	step config.StepConfig,
	builtins builtinContext,
) error {
	if step.Uses != config.BuiltinAutoCommit {
		return fmt.Errorf("%w: %q", errUnsupportedBuiltin, step.Uses)
	}

	if err := ralphgit.AutoCommit(ctx, ralphgit.AutoCommitOptions{
		WorkDir:           workDir,
		Mode:              builtins.Git.Commit,
		FallbackMessage:   builtins.Git.FallbackMessage,
		CommitMessagePath: builtins.CommitMessagePath,
		PRDPath:           builtins.PRDPath,
		BeforePRD:         builtins.BeforePRD,
	}); err != nil {
		return fmt.Errorf("run auto_commit builtin: %w", err)
	}

	return nil
}

func compilePhase(phaseName string, steps []config.StepConfig) ([]compiledStep, error) {
	compiled := make([]compiledStep, 0, len(steps))

	for _, step := range steps {
		expr, err := condition.Parse(step.If)
		if err != nil {
			return nil, fmt.Errorf(
				"parse if expression for phase=%s step=%s: %w",
				phaseName,
				step.Name,
				err,
			)
		}
		compiled = append(compiled, compiledStep{ConfigStep: step, Condition: expr})
	}

	return compiled, nil
}

func watchSignals(
	ctx context.Context,
	signals <-chan os.Signal,
	signalReceived *atomic.Bool,
	cancel context.CancelFunc,
) {
	select {
	case <-ctx.Done():
		return
	case <-signals:
		signalReceived.Store(true)
		cancel()
	}
}

func waitForNextIteration(
	ctx context.Context,
	sleepSeconds int,
	sleep func(time.Duration),
) bool {
	if sleepSeconds <= 0 {
		return ctx.Err() == nil
	}

	duration := time.Duration(sleepSeconds) * time.Second
	if sleep == nil {
		timer := time.NewTimer(duration)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return false
		case <-timer.C:
			return true
		}
	}

	done := make(chan struct{})
	go func() {
		sleep(duration)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return false
	case <-done:
		return true
	}
}

func resolveOptions(opts Options) Options {
	resolved := opts
	resolved.WorkDir = strings.TrimSpace(resolved.WorkDir)
	if resolved.WorkDir == "" {
		resolved.WorkDir = "."
	}

	resolved.ConfigPath = resolvePath(
		resolved.WorkDir,
		resolved.ConfigPath,
		ralphDirName,
		configFileName,
	)
	resolved.PromptPath = resolvePath(
		resolved.WorkDir,
		resolved.PromptPath,
		ralphDirName,
		promptFileName,
	)
	resolved.PRDPath = resolvePath(
		resolved.WorkDir,
		resolved.PRDPath,
		ralphDirName,
		prdFileName,
	)

	if resolved.Stdout == nil {
		resolved.Stdout = io.Discard
	}
	if resolved.Stderr == nil {
		resolved.Stderr = io.Discard
	}

	return resolved
}

func resolvePath(workDir string, path string, defaultSegments ...string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		parts := append([]string{workDir}, defaultSegments...)
		return filepath.Join(parts...)
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Join(workDir, trimmed)
}

func logStopLoop(stderr io.Writer, phaseName, stepName string, reason error) {
	if stderr == nil {
		return
	}

	reasonText := "unknown"
	if reason != nil {
		reasonText = strings.TrimSpace(reason.Error())
	}
	fmt.Fprintf(
		stderr,
		"[ralph] stop_loop phase=%s step=%s reason=%s\n",
		phaseName,
		stepName,
		reasonText,
	)
}

func logError(stderr io.Writer, err error) {
	if stderr == nil || err == nil {
		return
	}
	fmt.Fprintf(stderr, "[ralph] %s\n", strings.TrimSpace(err.Error()))
}
