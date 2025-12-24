package expression

import (
	"testing"
)

func TestTokenizer_NextToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
		wantErr  bool
	}{
		{
			name:  "simple function call",
			input: "success()",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "success"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "AND expression",
			input: "success() && failure()",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "success"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_AND, Literal: "&&"},
				{Type: TOKEN_IDENT, Literal: "failure"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "OR expression",
			input: "success() || failure()",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "success"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_OR, Literal: "||"},
				{Type: TOKEN_IDENT, Literal: "failure"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "NOT expression",
			input: "!failure()",
			expected: []Token{
				{Type: TOKEN_NOT, Literal: "!"},
				{Type: TOKEN_IDENT, Literal: "failure"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "env comparison with equals",
			input: "env.VAR == 'value'",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "env.VAR"},
				{Type: TOKEN_EQ, Literal: "=="},
				{Type: TOKEN_STRING, Literal: "value"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "env comparison with not equals",
			input: "env.VAR != 'value'",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "env.VAR"},
				{Type: TOKEN_NE, Literal: "!="},
				{Type: TOKEN_STRING, Literal: "value"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "contains function call",
			input: "contains(github.event.comment.body, '/deploy')",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "contains"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_IDENT, Literal: "github.event.comment.body"},
				{Type: TOKEN_COMMA, Literal: ","},
				{Type: TOKEN_STRING, Literal: "/deploy"},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "true literal",
			input: "true",
			expected: []Token{
				{Type: TOKEN_TRUE, Literal: "true"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "false literal",
			input: "false",
			expected: []Token{
				{Type: TOKEN_FALSE, Literal: "false"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "complex expression with parentheses",
			input: "(success() || failure()) && true",
			expected: []Token{
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_IDENT, Literal: "success"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_OR, Literal: "||"},
				{Type: TOKEN_IDENT, Literal: "failure"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_AND, Literal: "&&"},
				{Type: TOKEN_TRUE, Literal: "true"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "steps outcome",
			input: "steps.build.outcome == 'success'",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "steps.build.outcome"},
				{Type: TOKEN_EQ, Literal: "=="},
				{Type: TOKEN_STRING, Literal: "success"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "startsWith function",
			input: "startsWith(github.ref, 'refs/tags/')",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "startsWith"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_IDENT, Literal: "github.ref"},
				{Type: TOKEN_COMMA, Literal: ","},
				{Type: TOKEN_STRING, Literal: "refs/tags/"},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "hashFiles function",
			input: "hashFiles('**/package-lock.json') != ''",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "hashFiles"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_STRING, Literal: "**/package-lock.json"},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_NE, Literal: "!="},
				{Type: TOKEN_STRING, Literal: ""},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:    "unterminated string",
			input:   "'unterminated",
			wantErr: true,
		},
		{
			name:    "single ampersand",
			input:   "&",
			wantErr: true,
		},
		{
			name:    "single pipe",
			input:   "|",
			wantErr: true,
		},
		{
			name:    "single equals",
			input:   "=",
			wantErr: true,
		},
		{
			name:  "identifier with hyphen",
			input: "github.head-ref",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "github.head-ref"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "identifier with underscore",
			input: "env.MY_VAR",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "env.MY_VAR"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "always function",
			input: "always()",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "always"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
		{
			name:  "cancelled function",
			input: "cancelled()",
			expected: []Token{
				{Type: TOKEN_IDENT, Literal: "cancelled"},
				{Type: TOKEN_LPAREN, Literal: "("},
				{Type: TOKEN_RPAREN, Literal: ")"},
				{Type: TOKEN_EOF, Literal: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Tokenize() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Tokenize() unexpected error: %v", err)
				return
			}

			if len(tokens) != len(tt.expected) {
				t.Errorf("Tokenize() got %d tokens, want %d", len(tokens), len(tt.expected))
				t.Errorf("Got tokens: %v", tokens)
				return
			}

			for i, tok := range tokens {
				if tok.Type != tt.expected[i].Type {
					t.Errorf("Token[%d] type = %v, want %v", i, tok.Type, tt.expected[i].Type)
				}
				if tok.Literal != tt.expected[i].Literal {
					t.Errorf("Token[%d] literal = %q, want %q", i, tok.Literal, tt.expected[i].Literal)
				}
			}
		})
	}
}

func TestTokenType_String(t *testing.T) {
	tests := []struct {
		tokenType TokenType
		want      string
	}{
		{TOKEN_EOF, "EOF"},
		{TOKEN_AND, "AND"},
		{TOKEN_OR, "OR"},
		{TOKEN_NOT, "NOT"},
		{TOKEN_LPAREN, "LPAREN"},
		{TOKEN_RPAREN, "RPAREN"},
		{TOKEN_EQ, "EQ"},
		{TOKEN_NE, "NE"},
		{TOKEN_STRING, "STRING"},
		{TOKEN_IDENT, "IDENT"},
		{TOKEN_COMMA, "COMMA"},
		{TOKEN_TRUE, "TRUE"},
		{TOKEN_FALSE, "FALSE"},
		{TokenType(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.tokenType.String(); got != tt.want {
				t.Errorf("TokenType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTokenizer(t *testing.T) {
	tokenizer := NewTokenizer("test")
	if tokenizer == nil {
		t.Fatal("NewTokenizer() returned nil")
	}
	if tokenizer.input != "test" {
		t.Errorf("NewTokenizer() input = %q, want %q", tokenizer.input, "test")
	}
}
