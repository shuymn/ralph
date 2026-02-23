package ralphcondition

import "fmt"

const (
	ExitCodeValidation = 22

	ErrCodeConditionParse         = "CONDITION_PARSE"
	ErrCodeConditionUnknownSymbol = "CONDITION_UNKNOWN_SYMBOL"
)

type ParseError struct {
	code string
	msg  string
	pos  int
}

func (e *ParseError) Error() string {
	if e.pos < 0 {
		return e.msg
	}
	return fmt.Sprintf("%s (pos=%d)", e.msg, e.pos)
}

func (e *ParseError) Code() string {
	return e.code
}

func (e *ParseError) ExitCode() int {
	return ExitCodeValidation
}

func newParseError(msg string, pos int) error {
	return &ParseError{
		code: ErrCodeConditionParse,
		msg:  msg,
		pos:  pos,
	}
}

func newUnknownSymbolError(symbol string, pos int) error {
	return &ParseError{
		code: ErrCodeConditionUnknownSymbol,
		msg:  fmt.Sprintf("unknown symbol %q", symbol),
		pos:  pos,
	}
}

type Expr struct {
	root node
	raw  string
}

func Parse(input string) (Expr, error) {
	p := &parser{
		lexer: newLexer(input),
	}

	first, err := p.lexer.nextToken()
	if err != nil {
		return Expr{}, err
	}
	p.current = first

	root, err := p.parseExpression()
	if err != nil {
		return Expr{}, err
	}

	if p.current.typ != tokenEOF {
		return Expr{}, newParseError(
			"unexpected token "+p.current.value,
			p.current.pos,
		)
	}

	return Expr{
		root: root,
		raw:  input,
	}, nil
}

type parser struct {
	lexer   *lexer
	current token
}

func (p *parser) parseExpression() (node, error) {
	return p.parseOr()
}

func (p *parser) parseOr() (node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current.typ == tokenOr {
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: tokenOr, left: left, right: right}
	}

	return left, nil
}

func (p *parser) parseAnd() (node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.current.typ == tokenAnd {
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: tokenAnd, left: left, right: right}
	}

	return left, nil
}

func (p *parser) parseUnary() (node, error) {
	if p.current.typ == tokenNot {
		if err := p.next(); err != nil {
			return nil, err
		}
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unaryNode{expr: expr}, nil
	}

	return p.parsePrimary()
}

func (p *parser) parsePrimary() (node, error) {
	switch p.current.typ {
	case tokenTrue:
		if err := p.next(); err != nil {
			return nil, err
		}
		return boolNode{value: true}, nil
	case tokenFalse:
		if err := p.next(); err != nil {
			return nil, err
		}
		return boolNode{value: false}, nil
	case tokenLParen:
		if err := p.next(); err != nil {
			return nil, err
		}
		inside, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.current.typ != tokenRParen {
			return nil, newParseError("expected closing ')'", p.current.pos)
		}
		if err := p.next(); err != nil {
			return nil, err
		}
		return inside, nil
	case tokenIdent:
		return p.parseFunction()
	case tokenEOF:
		fallthrough
	case tokenRParen:
		fallthrough
	case tokenNot:
		fallthrough
	case tokenAnd:
		fallthrough
	case tokenOr:
		return nil, newParseError("unexpected token "+p.current.value, p.current.pos)
	}

	return nil, newParseError("unexpected token "+p.current.value, p.current.pos)
}

func (p *parser) parseFunction() (node, error) {
	name := p.current.value
	pos := p.current.pos
	if err := p.next(); err != nil {
		return nil, err
	}

	if p.current.typ != tokenLParen {
		return nil, newUnknownSymbolError(name, pos)
	}
	if err := p.next(); err != nil {
		return nil, err
	}

	if p.current.typ != tokenRParen {
		return nil, newParseError("functions do not accept arguments", p.current.pos)
	}
	if err := p.next(); err != nil {
		return nil, err
	}

	if !isSupportedFunction(name) {
		return nil, newUnknownSymbolError(name, pos)
	}

	return functionNode{name: name}, nil
}

func (p *parser) next() error {
	next, err := p.lexer.nextToken()
	if err != nil {
		return err
	}
	p.current = next
	return nil
}

func isSupportedFunction(name string) bool {
	switch name {
	case "always", "success", "failure", "changed":
		return true
	default:
		return false
	}
}
