package runner

import (
	"fmt"
	"io"

	"github.com/shuymn/ralph/internal/config"
	"github.com/shuymn/ralph/internal/prd"
)

func DryRun(opts Options) int {
	resolved, plan, err := loadExecutionPlan(opts)
	if err != nil {
		logError(resolved.Stderr, err)
		return ExitCodeConfigError
	}

	if err := renderDryRun(resolved.Stdout, resolved, plan); err != nil {
		logError(resolved.Stderr, err)
		return ExitCodeConfigError
	}

	return ExitCodeSuccess
}

func renderDryRun(out io.Writer, opts Options, plan executionPlan) error {
	if out == nil {
		return nil
	}

	if _, err := fmt.Fprintln(out, "[ralph] dry-run execution plan"); err != nil {
		return fmt.Errorf("write dry-run header: %w", err)
	}

	if err := writeDryRunPaths(out, opts); err != nil {
		return err
	}
	if err := writeDryRunAgent(out, plan.Config); err != nil {
		return err
	}
	if err := writeDryRunCompletion(out, plan.Config); err != nil {
		return err
	}
	if err := writeDryRunGit(out, plan.Config); err != nil {
		return err
	}
	if err := writeDryRunPRD(out, plan.PRD); err != nil {
		return err
	}
	if err := writeDryRunPhases(out, opts, plan); err != nil {
		return err
	}

	return nil
}

func writeDryRunPaths(out io.Writer, opts Options) error {
	if _, err := fmt.Fprintln(out, "paths:"); err != nil {
		return fmt.Errorf("write dry-run paths heading: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  workdir: %q\n", opts.WorkDir); err != nil {
		return fmt.Errorf("write dry-run workdir: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  config: %q\n", opts.ConfigPath); err != nil {
		return fmt.Errorf("write dry-run config path: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  prompt: %q\n", opts.PromptPath); err != nil {
		return fmt.Errorf("write dry-run prompt path: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  prd: %q\n", opts.PRDPath); err != nil {
		return fmt.Errorf("write dry-run prd path: %w", err)
	}

	return nil
}

func writeDryRunAgent(out io.Writer, cfg config.Config) error {
	if _, err := fmt.Fprintln(out, "agent:"); err != nil {
		return fmt.Errorf("write dry-run agent heading: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  command: %q\n", cfg.Agent.Command); err != nil {
		return fmt.Errorf("write dry-run agent command: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  max_iterations: %d\n", cfg.Agent.MaxIterations); err != nil {
		return fmt.Errorf("write dry-run max iterations: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  sleep_seconds: %d\n", cfg.Agent.SleepSeconds); err != nil {
		return fmt.Errorf("write dry-run sleep seconds: %w", err)
	}

	return nil
}

func writeDryRunCompletion(out io.Writer, cfg config.Config) error {
	if _, err := fmt.Fprintln(out, "completion:"); err != nil {
		return fmt.Errorf("write dry-run completion heading: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  strategy: %q\n", cfg.Completion.Strategy); err != nil {
		return fmt.Errorf("write dry-run completion strategy: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  signal: %q\n", cfg.Completion.Signal); err != nil {
		return fmt.Errorf("write dry-run completion signal: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  tail_lines: %d\n", cfg.Completion.TailLines); err != nil {
		return fmt.Errorf("write dry-run completion tail lines: %w", err)
	}

	return nil
}

func writeDryRunGit(out io.Writer, cfg config.Config) error {
	if _, err := fmt.Fprintln(out, "git:"); err != nil {
		return fmt.Errorf("write dry-run git heading: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  commit: %q\n", cfg.Git.Commit); err != nil {
		return fmt.Errorf("write dry-run git commit mode: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  fallback_message: %q\n", cfg.Git.FallbackMessage); err != nil {
		return fmt.Errorf("write dry-run git fallback message: %w", err)
	}

	return nil
}

func writeDryRunPRD(out io.Writer, doc prd.Document) error {
	if _, err := fmt.Fprintln(out, "prd:"); err != nil {
		return fmt.Errorf("write dry-run prd heading: %w", err)
	}
	if _, err := fmt.Fprintf(out, "  stories: %d\n", len(doc.Stories)); err != nil {
		return fmt.Errorf("write dry-run prd story count: %w", err)
	}

	return nil
}

func writeDryRunPhases(out io.Writer, opts Options, plan executionPlan) error {
	if _, err := fmt.Fprintln(out, "phases:"); err != nil {
		return fmt.Errorf("write dry-run phases heading: %w", err)
	}
	if err := writeDryRunPhase(out, phasePre, plan.PreSteps); err != nil {
		return err
	}
	if err := writeDryRunMainPhase(out, opts, plan.Config); err != nil {
		return err
	}
	if err := writeDryRunPhase(out, phasePost, plan.PostSteps); err != nil {
		return err
	}

	return nil
}

func writeDryRunPhase(out io.Writer, phaseName string, steps []compiledStep) error {
	if _, err := fmt.Fprintf(out, "  %s:\n", phaseName); err != nil {
		return fmt.Errorf("write dry-run %s heading: %w", phaseName, err)
	}
	if len(steps) == 0 {
		if _, err := fmt.Fprintln(out, "    steps: []"); err != nil {
			return fmt.Errorf("write dry-run %s empty steps: %w", phaseName, err)
		}
		return nil
	}

	if _, err := fmt.Fprintln(out, "    steps:"); err != nil {
		return fmt.Errorf("write dry-run %s steps heading: %w", phaseName, err)
	}

	for _, step := range steps {
		if _, err := fmt.Fprintf(out, "      - name: %q\n", step.ConfigStep.Name); err != nil {
			return fmt.Errorf("write dry-run %s step name: %w", phaseName, err)
		}
		if step.ConfigStep.Run != "" {
			if _, err := fmt.Fprintf(out, "        run: %q\n", step.ConfigStep.Run); err != nil {
				return fmt.Errorf("write dry-run %s step run: %w", phaseName, err)
			}
		}
		if step.ConfigStep.Uses != "" {
			if _, err := fmt.Fprintf(out, "        uses: %q\n", step.ConfigStep.Uses); err != nil {
				return fmt.Errorf("write dry-run %s step uses: %w", phaseName, err)
			}
		}
		if _, err := fmt.Fprintf(out, "        if: %q\n", step.ConfigStep.If); err != nil {
			return fmt.Errorf("write dry-run %s step if: %w", phaseName, err)
		}
		if _, err := fmt.Fprintf(out, "        on_fail: %q\n", step.ConfigStep.OnFail); err != nil {
			return fmt.Errorf("write dry-run %s step on_fail: %w", phaseName, err)
		}
	}

	return nil
}

func writeDryRunMainPhase(out io.Writer, opts Options, cfg config.Config) error {
	if _, err := fmt.Fprintln(out, "  main:"); err != nil {
		return fmt.Errorf("write dry-run main heading: %w", err)
	}
	if _, err := fmt.Fprintln(out, "    steps:"); err != nil {
		return fmt.Errorf("write dry-run main steps heading: %w", err)
	}
	if _, err := fmt.Fprintln(out, "      - name: \"main\""); err != nil {
		return fmt.Errorf("write dry-run main step name: %w", err)
	}
	if _, err := fmt.Fprintf(
		out,
		"        run: %q\n",
		fmt.Sprintf("%s %s %q", shellCommand, shellFlag, cfg.Agent.Command),
	); err != nil {
		return fmt.Errorf("write dry-run main run: %w", err)
	}
	if _, err := fmt.Fprintf(out, "        stdin: %q\n", opts.PromptPath); err != nil {
		return fmt.Errorf("write dry-run main stdin: %w", err)
	}
	if _, err := fmt.Fprintln(
		out,
		"        stdout: \"tempfile pattern ralph-main-*\"",
	); err != nil {
		return fmt.Errorf("write dry-run main stdout: %w", err)
	}

	return nil
}
