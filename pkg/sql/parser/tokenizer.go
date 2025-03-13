package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	// Token types
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenKeyword
	TokenString
	TokenNumber
	TokenPunctuation
	TokenOperator
	TokenWhitespace
	TokenComment
	TokenError
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Keywords is a map of SQL keywords to identify tokens
var Keywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "INTO": true,
	"VALUES": true, "CREATE": true, "COLLECTION": true, "DROP": true, "DELETE": true,
	"UPDATE": true, "SET": true, "AND": true, "OR": true, "NOT": true, "NULL": true,
	"TRUE": true, "FALSE": true, "COUNT": true, "NEAREST": true, "TO": true, "LIMIT": true,
	"USING": true, "METRIC": true, "JOIN": true, "ON": true, "AS": true, "ORDER": true, "BY": true,
	"ASC": true, "DESC": true, "GROUP": true, "HAVING": true, "DISTINCT": true, "UNION": true,
	"ALL": true, "IN": true, "EXISTS": true,
}

// Tokenizer breaks input into tokens
type Tokenizer struct {
	input     string
	pos       int
	start     int
	width     int
	tokens    []Token
	lastToken Token
}

// NewTokenizer creates a new tokenizer
func NewTokenizer(input string) *Tokenizer {
	return &Tokenizer{
		input:  input,
		tokens: make([]Token, 0),
	}
}

// Tokenize breaks the input string into tokens
func (t *Tokenizer) Tokenize() ([]Token, error) {
	// Reset state
	t.pos = 0
	t.start = 0
	t.width = 0
	t.tokens = make([]Token, 0)

	for state := lexText; state != nil; {
		state = state(t)
	}

	// Add EOF token
	t.tokens = append(t.tokens, Token{
		Type:  TokenEOF,
		Value: "",
		Pos:   t.pos,
	})

	return t.tokens, nil
}

// emit adds a token to the tokens slice
func (t *Tokenizer) emit(tokenType TokenType) {
	val := t.input[t.start:t.pos]
	t.tokens = append(t.tokens, Token{
		Type:  tokenType,
		Value: val,
		Pos:   t.start,
	})
	t.start = t.pos
}

// next advances the position in the input and returns the next rune
func (t *Tokenizer) next() rune {
	if t.pos >= len(t.input) {
		t.width = 0
		return 0
	}
	r := rune(t.input[t.pos])
	t.width = 1
	t.pos += t.width
	return r
}

// peek returns the next rune without advancing the position
func (t *Tokenizer) peek() rune {
	r := t.next()
	t.backup()
	return r
}

// backup steps back one rune
func (t *Tokenizer) backup() {
	t.pos -= t.width
}

// ignore skips the current token
func (t *Tokenizer) ignore() {
	t.start = t.pos
}

// error returns an error token and stops tokenizing
func (t *Tokenizer) error(format string, args ...interface{}) stateFn {
	t.tokens = append(t.tokens, Token{
		Type:  TokenError,
		Value: fmt.Sprintf(format, args...),
		Pos:   t.start,
	})
	return nil
}

// isWhitespace checks if a rune is whitespace
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// isAlphaNumeric checks if a rune is alphanumeric or underscore
func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// isDigit checks if a rune is a digit
func isDigit(r rune) bool {
	return unicode.IsDigit(r)
}

// stateFn represents a state function for the lexer
type stateFn func(*Tokenizer) stateFn

// lexText is the starting state function for the lexer
func lexText(t *Tokenizer) stateFn {
	for {
		r := t.peek()
		if r == 0 {
			return nil // End of input
		}

		switch {
		case isWhitespace(r):
			return lexWhitespace
		case r == '-':
			if t.pos+1 < len(t.input) && t.input[t.pos+1] == '-' {
				return lexComment
			}
			return lexOperator
		case r == '/':
			if t.pos+1 < len(t.input) && t.input[t.pos+1] == '*' {
				return lexMultiLineComment
			}
			return lexOperator
		case r == '\'':
			return lexString
		case r == '"':
			return lexQuotedIdentifier
		case r == '[':
			return lexVectorLiteral
		case unicode.IsLetter(r):
			return lexIdentifier
		case unicode.IsDigit(r):
			return lexNumber
		case r == ',' || r == '(' || r == ')' || r == ';' || r == '{' || r == '}':
			return lexPunctuation
		case r == '=' || r == '>' || r == '<' || r == '!' || r == '+' || r == '-' || r == '*' || r == '/' || r == '%':
			return lexOperator
		default:
			return t.error("unexpected character: %c", r)
		}
	}
}

// lexWhitespace tokenizes whitespace
func lexWhitespace(t *Tokenizer) stateFn {
	for isWhitespace(t.peek()) {
		t.next()
	}
	t.ignore() // ignore whitespace
	return lexText
}

// lexComment tokenizes single-line comments
func lexComment(t *Tokenizer) stateFn {
	// Skip '--'
	t.next()
	t.next()

	// Consume until end of line or input
	for {
		r := t.next()
		if r == '\n' || r == 0 {
			break
		}
	}
	t.ignore() // ignore comments
	return lexText
}

// lexMultiLineComment tokenizes multi-line comments
func lexMultiLineComment(t *Tokenizer) stateFn {
	// Skip '/*'
	t.next()
	t.next()

	// Consume until end of comment or input
	for {
		if t.next() == '*' && t.peek() == '/' {
			t.next() // consume the '/'
			break
		}
		if t.peek() == 0 {
			return t.error("unclosed comment")
		}
	}
	t.ignore() // ignore comments
	return lexText
}

// lexString tokenizes string literals
func lexString(t *Tokenizer) stateFn {
	// Skip opening quote
	t.next()

	// Consume until closing quote or input
	for {
		r := t.next()
		if r == 0 {
			return t.error("unclosed string literal")
		}
		if r == '\'' {
			break
		}
		if r == '\\' && t.peek() == '\'' {
			t.next() // skip the escaped quote
		}
	}
	t.emit(TokenString)
	return lexText
}

// lexQuotedIdentifier tokenizes quoted identifiers
func lexQuotedIdentifier(t *Tokenizer) stateFn {
	// Skip opening quote
	t.next()

	// Consume until closing quote or input
	for {
		r := t.next()
		if r == 0 {
			return t.error("unclosed quoted identifier")
		}
		if r == '"' {
			break
		}
		if r == '\\' && t.peek() == '"' {
			t.next() // skip the escaped quote
		}
	}
	t.emit(TokenIdentifier)
	return lexText
}

// lexVectorLiteral tokenizes vector literals like [1.0, 2.0, 3.0]
func lexVectorLiteral(t *Tokenizer) stateFn {
	// Skip opening bracket
	t.next()

	// Consume until closing bracket or input
	depth := 1
	for {
		r := t.next()
		if r == 0 {
			return t.error("unclosed vector literal")
		}
		if r == '[' {
			depth++
		}
		if r == ']' {
			depth--
			if depth == 0 {
				break
			}
		}
	}
	t.emit(TokenString) // We'll handle the vector parsing later
	return lexText
}

// lexIdentifier tokenizes identifiers and keywords
func lexIdentifier(t *Tokenizer) stateFn {
	for isAlphaNumeric(t.peek()) {
		t.next()
	}
	
	// Check if it's a keyword
	value := strings.ToUpper(t.input[t.start:t.pos])
	if Keywords[value] {
		t.emit(TokenKeyword)
	} else {
		t.emit(TokenIdentifier)
	}
	
	return lexText
}

// lexNumber tokenizes numeric literals
func lexNumber(t *Tokenizer) stateFn {
	// Digits before decimal point
	for isDigit(t.peek()) {
		t.next()
	}
	
	// Decimal point and digits after
	if t.peek() == '.' {
		t.next()
		hasDigitsAfterPoint := false
		for isDigit(t.peek()) {
			t.next()
			hasDigitsAfterPoint = true
		}
		if !hasDigitsAfterPoint {
			return t.error("expected digit after decimal point")
		}
	}
	
	// Scientific notation
	if t.peek() == 'e' || t.peek() == 'E' {
		t.next()
		if t.peek() == '+' || t.peek() == '-' {
			t.next()
		}
		hasDigitsInExponent := false
		for isDigit(t.peek()) {
			t.next()
			hasDigitsInExponent = true
		}
		if !hasDigitsInExponent {
			return t.error("expected digit in exponent")
		}
	}
	
	t.emit(TokenNumber)
	return lexText
}

// lexPunctuation tokenizes punctuation
func lexPunctuation(t *Tokenizer) stateFn {
	t.next()
	t.emit(TokenPunctuation)
	return lexText
}

// lexOperator tokenizes operators
func lexOperator(t *Tokenizer) stateFn {
	// Single character operators
	t.next()
	
	// Multi-character operators
	switch t.input[t.pos-1] {
	case '=', '!', '<', '>':
		if t.peek() == '=' {
			t.next()
		}
	}
	
	t.emit(TokenOperator)
	return lexText
} 