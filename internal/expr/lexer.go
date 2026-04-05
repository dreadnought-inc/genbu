package expr

import (
	"fmt"
	"unicode"
)

type tokenType int

const (
	tokenIdent  tokenType = iota // function name or variable reference
	tokenString                  // "..." or '...'
	tokenNumber                  // integer literal
	tokenLParen                  // (
	tokenRParen                  // )
	tokenComma                   // ,
	tokenEOF
)

type token struct {
	typ tokenType
	val string
}

type lexer struct {
	input []rune
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: []rune(input)}
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *lexer) next() rune {
	r := l.peek()
	l.pos++
	return r
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *lexer) tokenize() ([]token, error) {
	var tokens []token

	for {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			tokens = append(tokens, token{typ: tokenEOF})
			return tokens, nil
		}

		r := l.peek()

		switch {
		case r == '(':
			l.next()
			tokens = append(tokens, token{typ: tokenLParen, val: "("})
		case r == ')':
			l.next()
			tokens = append(tokens, token{typ: tokenRParen, val: ")"})
		case r == ',':
			l.next()
			tokens = append(tokens, token{typ: tokenComma, val: ","})
		case r == '"' || r == '\'':
			tok, err := l.readString(r)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		case unicode.IsDigit(r):
			tokens = append(tokens, l.readNumber())
		case r == '_' || unicode.IsLetter(r):
			tokens = append(tokens, l.readIdent())
		default:
			return nil, fmt.Errorf("unexpected character %q at position %d", string(r), l.pos)
		}
	}
}

func (l *lexer) readString(quote rune) (token, error) {
	l.next() // skip opening quote
	start := l.pos

	for l.pos < len(l.input) {
		r := l.next()
		if r == '\\' && l.pos < len(l.input) {
			l.next() // skip escaped char
			continue
		}
		if r == quote {
			return token{typ: tokenString, val: string(l.input[start : l.pos-1])}, nil
		}
	}

	return token{}, fmt.Errorf("unterminated string starting at position %d", start-1)
}

func (l *lexer) readNumber() token {
	start := l.pos
	for l.pos < len(l.input) && unicode.IsDigit(l.input[l.pos]) {
		l.pos++
	}
	return token{typ: tokenNumber, val: string(l.input[start:l.pos])}
}

func (l *lexer) readIdent() token {
	start := l.pos
	for l.pos < len(l.input) && (l.input[l.pos] == '_' || unicode.IsLetter(l.input[l.pos]) || unicode.IsDigit(l.input[l.pos])) {
		l.pos++
	}
	return token{typ: tokenIdent, val: string(l.input[start:l.pos])}
}
