package ralphcondition

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

const ErrCodeConditionRuntime = "CONDITION_RUNTIME"

type RuntimeError struct {
	code  string
	msg   string
	cause error
}

func (e *RuntimeError) Error() string {
	if e.cause == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %v", e.msg, e.cause)
}

func (e *RuntimeError) Unwrap() error {
	return e.cause
}

func (e *RuntimeError) Code() string {
	return e.code
}

func (e *RuntimeError) ExitCode() int {
	return ExitCodeValidation
}

func newRuntimeError(msg string, cause error) error {
	return &RuntimeError{
		code:  ErrCodeConditionRuntime,
		msg:   msg,
		cause: cause,
	}
}

type ChangedFunc func() (bool, error)

type Context struct {
	Success bool
	Failure bool
	Changed ChangedFunc
}

func NewContext(success, failure bool, changed ChangedFunc) Context {
	return Context{
		Success: success,
		Failure: failure,
		Changed: changed,
	}
}

func NewGitChangedFunc(workingDir string) ChangedFunc {
	return func() (bool, error) {
		cmd := exec.CommandContext(context.Background(), "git", "status", "--porcelain")
		cmd.Dir = workingDir

		stdout, err := cmd.Output()
		if err != nil {
			return false, newRuntimeError(
				"changed() failed to run git status --porcelain",
				err,
			)
		}

		return len(bytes.TrimSpace(stdout)) > 0, nil
	}
}

func Eval(input string, ctx Context) (bool, error) {
	expr, err := Parse(input)
	if err != nil {
		return false, err
	}
	return expr.Eval(ctx)
}

func (e Expr) Eval(ctx Context) (bool, error) {
	if e.root == nil {
		return false, newRuntimeError(
			"expression is not initialized",
			nil,
		)
	}

	return e.root.eval(ctx)
}

type node interface {
	eval(ctx Context) (bool, error)
}

type boolNode struct {
	value bool
}

func (n boolNode) eval(_ Context) (bool, error) {
	return n.value, nil
}

type unaryNode struct {
	expr node
}

func (n unaryNode) eval(ctx Context) (bool, error) {
	result, err := n.expr.eval(ctx)
	if err != nil {
		return false, err
	}
	return !result, nil
}

type binaryNode struct {
	op    tokenType
	left  node
	right node
}

func (n binaryNode) eval(ctx Context) (bool, error) {
	switch n.op {
	case tokenAnd:
		left, err := n.left.eval(ctx)
		if err != nil {
			return false, err
		}
		if !left {
			return false, nil
		}
		return n.right.eval(ctx)
	case tokenOr:
		left, err := n.left.eval(ctx)
		if err != nil {
			return false, err
		}
		if left {
			return true, nil
		}
		return n.right.eval(ctx)
	case tokenEOF:
		fallthrough
	case tokenIdent:
		fallthrough
	case tokenTrue:
		fallthrough
	case tokenFalse:
		fallthrough
	case tokenLParen:
		fallthrough
	case tokenRParen:
		fallthrough
	case tokenNot:
		return false, newRuntimeError(
			"unsupported binary operator",
			nil,
		)
	}

	return false, newRuntimeError(
		"unsupported binary operator",
		nil,
	)
}

type functionNode struct {
	name string
}

func (n functionNode) eval(ctx Context) (bool, error) {
	switch n.name {
	case "always":
		return true, nil
	case "success":
		return ctx.Success, nil
	case "failure":
		return ctx.Failure, nil
	case "changed":
		if ctx.Changed == nil {
			return false, newRuntimeError(
				"changed() callback is not configured",
				nil,
			)
		}
		changed, err := ctx.Changed()
		if err != nil {
			var runtimeErr *RuntimeError
			if errors.As(err, &runtimeErr) {
				return false, err
			}
			return false, newRuntimeError(
				"changed() callback failed",
				err,
			)
		}
		return changed, nil
	default:
		return false, newUnknownSymbolError(n.name, -1)
	}
}
