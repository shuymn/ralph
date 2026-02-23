package ralphcondition

import (
	"unicode"
	"unicode/utf8"
)

type tokenType string

const (
	tokenEOF    tokenType = "EOF"
	tokenIdent  tokenType = "IDENT"
	tokenTrue   tokenType = "TRUE"
	tokenFalse  tokenType = "FALSE"
	tokenLParen tokenType = "LPAREN"
	tokenRParen tokenType = "RPAREN"
	tokenNot    tokenType = "NOT"
	tokenAnd    tokenType = "AND"
	tokenOr     tokenType = "OR"
)

type token struct {
	typ   tokenType
	value string
	pos   int
}

type lexer struct {
	input string
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{
		input: input,
	}
}

func (l *lexer) nextToken() (token, error) {
	l.skipSpaces()
	start := l.pos

	if l.pos >= len(l.input) {
		return token{typ: tokenEOF, pos: start}, nil
	}

	switch l.peek() {
	case '(':
		l.advance()
		return token{typ: tokenLParen, value: "(", pos: start}, nil
	case ')':
		l.advance()
		return token{typ: tokenRParen, value: ")", pos: start}, nil
	case '!':
		l.advance()
		return token{typ: tokenNot, value: "!", pos: start}, nil
	case '&':
		l.advance()
		if l.peek() != '&' {
			return token{}, newParseError("expected '&' after '&'", start)
		}
		l.advance()
		return token{typ: tokenAnd, value: "&&", pos: start}, nil
	case '|':
		l.advance()
		if l.peek() != '|' {
			return token{}, newParseError("expected '|' after '|'", start)
		}
		l.advance()
		return token{typ: tokenOr, value: "||", pos: start}, nil
	default:
		if isIdentStart(l.peek()) {
			ident := l.readIdent()
			switch ident {
			case "true":
				return token{typ: tokenTrue, value: ident, pos: start}, nil
			case "false":
				return token{typ: tokenFalse, value: ident, pos: start}, nil
			default:
				return token{typ: tokenIdent, value: ident, pos: start}, nil
			}
		}

		ch := l.peek()
		return token{}, newParseError("unsupported character '"+string(ch)+"'", start)
	}
}

func (l *lexer) skipSpaces() {
	for l.pos < len(l.input) {
		if !unicode.IsSpace(l.peek()) {
			return
		}
		l.advance()
	}
}

func (l *lexer) readIdent() string {
	start := l.pos
	for l.pos < len(l.input) {
		if !isIdentContinue(l.peek()) {
			break
		}
		l.advance()
	}
	return l.input[start:l.pos]
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return rune(0)
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *lexer) advance() {
	_, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += size
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentContinue(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
}
