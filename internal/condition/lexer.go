package condition

import (
	"fmt"
	"unicode"
)

type tokenType string

const (
	tokenEOF        tokenType = "eof"
	tokenIdentifier tokenType = "identifier"
	tokenTrue       tokenType = "true"
	tokenFalse      tokenType = "false"
	tokenLParen     tokenType = "("
	tokenRParen     tokenType = ")"
	tokenNot        tokenType = "!"
	tokenAnd        tokenType = "&&"
	tokenOr         tokenType = "||"
)

type token struct {
	typ      tokenType
	value    string
	position int
}

func lex(input string) ([]token, error) {
	runes := []rune(input)
	tokens := make([]token, 0, len(runes)+1)

	for index := 0; index < len(runes); {
		ch := runes[index]
		if unicode.IsSpace(ch) {
			index++
			continue
		}

		switch ch {
		case '(':
			tokens = append(tokens, token{typ: tokenLParen, value: "(", position: index})
			index++
		case ')':
			tokens = append(tokens, token{typ: tokenRParen, value: ")", position: index})
			index++
		case '!':
			tokens = append(tokens, token{typ: tokenNot, value: "!", position: index})
			index++
		case '&':
			if index+1 >= len(runes) || runes[index+1] != '&' {
				return nil, newParseError(
					fmt.Sprintf("invalid token %q at position %d", ch, index),
				)
			}
			tokens = append(tokens, token{typ: tokenAnd, value: "&&", position: index})
			index += 2
		case '|':
			if index+1 >= len(runes) || runes[index+1] != '|' {
				return nil, newParseError(
					fmt.Sprintf("invalid token %q at position %d", ch, index),
				)
			}
			tokens = append(tokens, token{typ: tokenOr, value: "||", position: index})
			index += 2
		default:
			if !isIdentifierStart(ch) {
				return nil, newParseError(
					fmt.Sprintf("invalid token %q at position %d", ch, index),
				)
			}

			start := index
			for index < len(runes) && isIdentifierPart(runes[index]) {
				index++
			}
			value := string(runes[start:index])
			kind := tokenIdentifier
			switch value {
			case "true":
				kind = tokenTrue
			case "false":
				kind = tokenFalse
			}
			tokens = append(tokens, token{typ: kind, value: value, position: start})
		}
	}

	tokens = append(tokens, token{typ: tokenEOF, value: "", position: len(runes)})
	return tokens, nil
}

func isIdentifierStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentifierPart(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}
