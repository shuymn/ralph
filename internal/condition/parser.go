package condition

import (
	"fmt"
	"strings"
)

type Expression struct {
	root node
}

type parser struct {
	tokens  []token
	current int
}

func Parse(raw string) (Expression, error) {
	if strings.TrimSpace(raw) == "" {
		return Expression{}, newParseError("expression is required")
	}

	tokens, err := lex(raw)
	if err != nil {
		return Expression{}, err
	}

	p := &parser{tokens: tokens}
	root, err := p.parseOr()
	if err != nil {
		return Expression{}, err
	}

	if !p.match(tokenEOF) {
		next := p.peek()
		return Expression{}, newParseError(
			fmt.Sprintf("unexpected token %q at position %d", next.value, next.position),
		)
	}

	return Expression{root: root}, nil
}

func (p *parser) parseOr() (node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.match(tokenOr) {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = binaryNode{operator: tokenOr, left: left, right: right}
	}

	return left, nil
}

func (p *parser) parseAnd() (node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.match(tokenAnd) {
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = binaryNode{operator: tokenAnd, left: left, right: right}
	}

	return left, nil
}

func (p *parser) parseUnary() (node, error) {
	if p.match(tokenNot) {
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unaryNode{operator: tokenNot, expr: expr}, nil
	}

	return p.parsePrimary()
}

func (p *parser) parsePrimary() (node, error) {
	next := p.advance()

	if next.typ == tokenTrue {
		return literalNode{value: true}, nil
	}

	if next.typ == tokenFalse {
		return literalNode{value: false}, nil
	}

	if next.typ == tokenIdentifier {
		name := next.value
		if !p.match(tokenLParen) {
			return nil, newParseError(fmt.Sprintf("unknown symbol %q", name))
		}
		if !p.match(tokenRParen) {
			return nil, newParseError(
				fmt.Sprintf("function %q does not accept arguments", name),
			)
		}
		if !isSupportedFunction(name) {
			return nil, newParseError(fmt.Sprintf("unknown symbol %q", name))
		}
		return functionNode{name: name}, nil
	}

	if next.typ == tokenLParen {
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if !p.match(tokenRParen) {
			found := p.peek()
			return nil, newParseError(
				fmt.Sprintf("expected ')' at position %d, found %q", found.position, found.value),
			)
		}
		return groupingNode{expr: expr}, nil
	}

	if next.typ == tokenEOF {
		return nil, newParseError("unexpected end of expression")
	}

	return nil, newParseError(
		fmt.Sprintf("unexpected token %q at position %d", next.value, next.position),
	)
}

func (p *parser) match(kind tokenType) bool {
	if p.peek().typ != kind {
		return false
	}
	p.current++
	return true
}

func (p *parser) peek() token {
	if p.current >= len(p.tokens) {
		return token{typ: tokenEOF, value: "", position: len(p.tokens)}
	}
	return p.tokens[p.current]
}

func (p *parser) advance() token {
	next := p.peek()
	if p.current < len(p.tokens) {
		p.current++
	}
	return next
}

func isSupportedFunction(name string) bool {
	switch name {
	case functionAlways, functionSuccess, functionFailure, functionChanged:
		return true
	default:
		return false
	}
}
