package condition

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type ErrorKind string

const (
	ErrorKindParse      ErrorKind = "parse"
	ErrorKindEvaluation ErrorKind = "evaluation"
)

const (
	functionAlways  = "always"
	functionSuccess = "success"
	functionFailure = "failure"
	functionChanged = "changed"
)

const (
	gitCommand        = "git"
	gitStatusArgument = "status"
	gitPorcelainFlag  = "--porcelain"
)

type Context struct {
	Success bool
	Changed ChangedFunc
}

type ChangedFunc func() (bool, error)

type Error struct {
	kind ErrorKind
	msg  string
	err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %v", e.msg, e.err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *Error) Kind() ErrorKind {
	if e == nil {
		return ""
	}
	return e.kind
}

func IsParseError(err error) bool {
	var typed *Error
	if !errors.As(err, &typed) {
		return false
	}
	return typed.Kind() == ErrorKindParse
}

func IsEvaluationError(err error) bool {
	var typed *Error
	if !errors.As(err, &typed) {
		return false
	}
	return typed.Kind() == ErrorKindEvaluation
}

func Evaluate(expression string, context Context) (bool, error) {
	parsed, err := Parse(expression)
	if err != nil {
		return false, err
	}
	return parsed.Evaluate(context)
}

func (e Expression) Evaluate(context Context) (bool, error) {
	if e.root == nil {
		return false, newParseError("expression is required")
	}
	return e.root.evaluate(context)
}

func GitChanged(workDir string) ChangedFunc {
	return func() (bool, error) {
		cmd := exec.CommandContext(
			context.Background(),
			gitCommand,
			gitStatusArgument,
			gitPorcelainFlag,
		)
		if workDir != "" {
			cmd.Dir = workDir
		}

		output, err := cmd.Output()
		if err != nil {
			return false, fmt.Errorf(
				"run %s %s %s: %w",
				gitCommand,
				gitStatusArgument,
				gitPorcelainFlag,
				err,
			)
		}
		return len(bytes.TrimSpace(output)) > 0, nil
	}
}

type node interface {
	evaluate(context Context) (bool, error)
}

type literalNode struct {
	value bool
}

func (n literalNode) evaluate(_ Context) (bool, error) {
	return n.value, nil
}

type groupingNode struct {
	expr node
}

func (n groupingNode) evaluate(context Context) (bool, error) {
	return n.expr.evaluate(context)
}

type unaryNode struct {
	operator tokenType
	expr     node
}

func (n unaryNode) evaluate(context Context) (bool, error) {
	value, err := n.expr.evaluate(context)
	if err != nil {
		return false, err
	}

	if n.operator != tokenNot {
		return false, newEvaluationError(
			fmt.Sprintf("unsupported unary operator %q", n.operator),
			nil,
		)
	}
	return !value, nil
}

type binaryNode struct {
	operator tokenType
	left     node
	right    node
}

func (n binaryNode) evaluate(context Context) (bool, error) {
	if n.operator == tokenAnd {
		left, err := n.left.evaluate(context)
		if err != nil {
			return false, err
		}
		if !left {
			return false, nil
		}
		return n.right.evaluate(context)
	}

	if n.operator == tokenOr {
		left, err := n.left.evaluate(context)
		if err != nil {
			return false, err
		}
		if left {
			return true, nil
		}
		return n.right.evaluate(context)
	}

	return false, newEvaluationError(
		fmt.Sprintf("unsupported binary operator %q", n.operator),
		nil,
	)
}

type functionNode struct {
	name string
}

func (n functionNode) evaluate(context Context) (bool, error) {
	switch n.name {
	case functionAlways:
		return true, nil
	case functionSuccess:
		return context.Success, nil
	case functionFailure:
		return !context.Success, nil
	case functionChanged:
		return context.evaluateChanged()
	default:
		return false, newEvaluationError(fmt.Sprintf("unknown function %q", n.name), nil)
	}
}

func (c Context) evaluateChanged() (bool, error) {
	changed := c.Changed
	if changed == nil {
		changed = GitChanged("")
	}

	value, err := changed()
	if err != nil {
		return false, newEvaluationError("evaluate changed()", err)
	}
	return value, nil
}

func newParseError(msg string) error {
	return &Error{
		kind: ErrorKindParse,
		msg:  strings.TrimSpace(msg),
	}
}

func newEvaluationError(msg string, err error) error {
	return &Error{
		kind: ErrorKindEvaluation,
		msg:  strings.TrimSpace(msg),
		err:  err,
	}
}
