package expression

import (
	"fmt"
	"unicode"
)

// TokenType represents the type of a token.
type TokenType int

const (
	TOKEN_EOF    TokenType = iota
	TOKEN_AND              // &&
	TOKEN_OR               // ||
	TOKEN_NOT              // !
	TOKEN_LPAREN           // (
	TOKEN_RPAREN           // )
	TOKEN_EQ               // ==
	TOKEN_NE               // !=
	TOKEN_STRING           // 'value'
	TOKEN_IDENT            // env.VAR, steps.id.outcome, function names
	TOKEN_COMMA            // ,
	TOKEN_TRUE             // true
	TOKEN_FALSE            // false
)

// String returns a string representation of the token type.
func (t TokenType) String() string {
	switch t {
	case TOKEN_EOF:
		return "EOF"
	case TOKEN_AND:
		return "AND"
	case TOKEN_OR:
		return "OR"
	case TOKEN_NOT:
		return "NOT"
	case TOKEN_LPAREN:
		return "LPAREN"
	case TOKEN_RPAREN:
		return "RPAREN"
	case TOKEN_EQ:
		return "EQ"
	case TOKEN_NE:
		return "NE"
	case TOKEN_STRING:
		return "STRING"
	case TOKEN_IDENT:
		return "IDENT"
	case TOKEN_COMMA:
		return "COMMA"
	case TOKEN_TRUE:
		return "TRUE"
	case TOKEN_FALSE:
		return "FALSE"
	default:
		return "UNKNOWN"
	}
}

// Token represents a lexical token.
type Token struct {
	Type    TokenType
	Literal string
	Pos     int // Position in the input string
}

// Tokenizer performs lexical analysis on expression strings.
type Tokenizer struct {
	input   string
	pos     int  // current position in input
	readPos int  // reading position (after current char)
	ch      byte // current char under examination
}

// NewTokenizer creates a new Tokenizer for the given input string.
func NewTokenizer(input string) *Tokenizer {
	t := &Tokenizer{input: input}
	t.readChar()
	return t
}

// readChar advances the position and reads the next character.
func (t *Tokenizer) readChar() {
	if t.readPos >= len(t.input) {
		t.ch = 0 // ASCII NUL = end of input
	} else {
		t.ch = t.input[t.readPos]
	}
	t.pos = t.readPos
	t.readPos++
}

// peekChar returns the next character without advancing the position.
func (t *Tokenizer) peekChar() byte {
	if t.readPos >= len(t.input) {
		return 0
	}
	return t.input[t.readPos]
}

// skipWhitespace skips any whitespace characters.
func (t *Tokenizer) skipWhitespace() {
	for t.ch == ' ' || t.ch == '\t' || t.ch == '\n' || t.ch == '\r' {
		t.readChar()
	}
}

// NextToken returns the next token from the input.
func (t *Tokenizer) NextToken() (Token, error) {
	t.skipWhitespace()

	tok := Token{Pos: t.pos}

	switch t.ch {
	case 0:
		tok.Type = TOKEN_EOF
		tok.Literal = ""
	case '&':
		if t.peekChar() == '&' {
			pos := t.pos
			t.readChar()
			tok.Type = TOKEN_AND
			tok.Literal = "&&"
			tok.Pos = pos
		} else {
			return tok, fmt.Errorf("unexpected character '%c' at position %d, expected '&&'", t.ch, t.pos)
		}
		t.readChar()
	case '|':
		if t.peekChar() == '|' {
			pos := t.pos
			t.readChar()
			tok.Type = TOKEN_OR
			tok.Literal = "||"
			tok.Pos = pos
		} else {
			return tok, fmt.Errorf("unexpected character '%c' at position %d, expected '||'", t.ch, t.pos)
		}
		t.readChar()
	case '!':
		if t.peekChar() == '=' {
			pos := t.pos
			t.readChar()
			tok.Type = TOKEN_NE
			tok.Literal = "!="
			tok.Pos = pos
			t.readChar()
		} else {
			tok.Type = TOKEN_NOT
			tok.Literal = "!"
			t.readChar()
		}
	case '=':
		if t.peekChar() == '=' {
			pos := t.pos
			t.readChar()
			tok.Type = TOKEN_EQ
			tok.Literal = "=="
			tok.Pos = pos
		} else {
			return tok, fmt.Errorf("unexpected character '%c' at position %d, expected '=='", t.ch, t.pos)
		}
		t.readChar()
	case '(':
		tok.Type = TOKEN_LPAREN
		tok.Literal = "("
		t.readChar()
	case ')':
		tok.Type = TOKEN_RPAREN
		tok.Literal = ")"
		t.readChar()
	case ',':
		tok.Type = TOKEN_COMMA
		tok.Literal = ","
		t.readChar()
	case '\'':
		str, err := t.readString()
		if err != nil {
			return tok, err
		}
		tok.Type = TOKEN_STRING
		tok.Literal = str
	default:
		if isIdentStart(t.ch) {
			ident := t.readIdentifier()
			// Check for keywords
			switch ident {
			case "true":
				tok.Type = TOKEN_TRUE
			case "false":
				tok.Type = TOKEN_FALSE
			default:
				tok.Type = TOKEN_IDENT
			}
			tok.Literal = ident
		} else {
			return tok, fmt.Errorf("unexpected character '%c' at position %d", t.ch, t.pos)
		}
	}

	return tok, nil
}

// readString reads a string literal enclosed in single quotes.
func (t *Tokenizer) readString() (string, error) {
	startPos := t.pos
	t.readChar() // skip opening quote

	var result []byte
	for t.ch != '\'' && t.ch != 0 {
		result = append(result, t.ch)
		t.readChar()
	}

	if t.ch == 0 {
		return "", fmt.Errorf("unterminated string starting at position %d", startPos)
	}

	t.readChar() // skip closing quote
	return string(result), nil
}

// readIdentifier reads an identifier (including dots for property access).
func (t *Tokenizer) readIdentifier() string {
	startPos := t.pos
	for isIdentChar(t.ch) {
		t.readChar()
	}
	return t.input[startPos:t.pos]
}

// isIdentStart returns true if ch can start an identifier.
func isIdentStart(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

// isIdentChar returns true if ch can be part of an identifier.
func isIdentChar(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' || ch == '.' || ch == '-'
}

// Tokenize tokenizes the entire input and returns all tokens.
func Tokenize(input string) ([]Token, error) {
	tokenizer := NewTokenizer(input)
	var tokens []Token

	for {
		tok, err := tokenizer.NextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TOKEN_EOF {
			break
		}
	}

	return tokens, nil
}
