package expression

import (
	"fmt"
)

// Parser parses tokens into an Abstract Syntax Tree.
type Parser struct {
	tokenizer *Tokenizer
	curToken  Token
	peekToken Token
	errors    []string
}

// NewParser creates a new Parser for the given input string.
func NewParser(input string) *Parser {
	p := &Parser{
		tokenizer: NewTokenizer(input),
	}
	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()
	return p
}

// nextToken advances to the next token.
func (p *Parser) nextToken() error {
	p.curToken = p.peekToken
	tok, err := p.tokenizer.NextToken()
	if err != nil {
		p.errors = append(p.errors, err.Error())
		return err
	}
	p.peekToken = tok
	return nil
}

// Parse parses the input and returns the root node of the AST.
func (p *Parser) Parse() (Node, error) {
	node := p.parseExpression()
	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse errors: %v", p.errors)
	}
	return node, nil
}

// parseExpression parses an expression (handles OR - lowest precedence).
func (p *Parser) parseExpression() Node {
	return p.parseOrExpr()
}

// parseOrExpr parses OR expressions (||).
func (p *Parser) parseOrExpr() Node {
	left := p.parseAndExpr()

	for p.curToken.Type == TOKEN_OR {
		operator := p.curToken.Type
		p.nextToken()
		right := p.parseAndExpr()
		left = &BinaryExpr{
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}

	return left
}

// parseAndExpr parses AND expressions (&&).
func (p *Parser) parseAndExpr() Node {
	left := p.parseUnaryExpr()

	for p.curToken.Type == TOKEN_AND {
		operator := p.curToken.Type
		p.nextToken()
		right := p.parseUnaryExpr()
		left = &BinaryExpr{
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}

	return left
}

// parseUnaryExpr parses unary expressions (!).
func (p *Parser) parseUnaryExpr() Node {
	if p.curToken.Type == TOKEN_NOT {
		operator := p.curToken.Type
		p.nextToken()
		operand := p.parseUnaryExpr()
		return &UnaryExpr{
			Operator: operator,
			Operand:  operand,
		}
	}
	return p.parseComparisonExpr()
}

// parseComparisonExpr parses comparison expressions (==, !=).
func (p *Parser) parseComparisonExpr() Node {
	left := p.parsePrimaryExpr()

	if p.curToken.Type == TOKEN_EQ || p.curToken.Type == TOKEN_NE {
		operator := p.curToken.Type
		p.nextToken()
		right := p.parsePrimaryExpr()
		return &BinaryExpr{
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}

	return left
}

// parsePrimaryExpr parses primary expressions (literals, identifiers, function calls, grouped expressions).
func (p *Parser) parsePrimaryExpr() Node {
	switch p.curToken.Type {
	case TOKEN_TRUE:
		p.nextToken()
		return &BoolLiteral{Value: true}

	case TOKEN_FALSE:
		p.nextToken()
		return &BoolLiteral{Value: false}

	case TOKEN_STRING:
		lit := &StringLiteral{Value: p.curToken.Literal}
		p.nextToken()
		return lit

	case TOKEN_IDENT:
		return p.parseIdentOrCall()

	case TOKEN_LPAREN:
		p.nextToken() // consume '('
		expr := p.parseExpression()
		if p.curToken.Type != TOKEN_RPAREN {
			p.errors = append(p.errors, fmt.Sprintf("expected ')' but got %s", p.curToken.Type))
			return nil
		}
		p.nextToken() // consume ')'
		return expr

	default:
		p.errors = append(p.errors, fmt.Sprintf("unexpected token: %s", p.curToken.Type))
		return nil
	}
}

// parseIdentOrCall parses an identifier or function call.
func (p *Parser) parseIdentOrCall() Node {
	name := p.curToken.Literal
	p.nextToken()

	// Check if it's a function call
	if p.curToken.Type == TOKEN_LPAREN {
		p.nextToken() // consume '('
		args := p.parseArguments()
		if p.curToken.Type != TOKEN_RPAREN {
			p.errors = append(p.errors, fmt.Sprintf("expected ')' but got %s", p.curToken.Type))
			return nil
		}
		p.nextToken() // consume ')'
		return &CallExpr{
			FuncName:  name,
			Arguments: args,
		}
	}

	return &Identifier{Value: name}
}

// parseArguments parses function arguments.
func (p *Parser) parseArguments() []Node {
	var args []Node

	// Empty argument list
	if p.curToken.Type == TOKEN_RPAREN {
		return args
	}

	// First argument
	args = append(args, p.parseExpression())

	// Additional arguments
	for p.curToken.Type == TOKEN_COMMA {
		p.nextToken() // consume ','
		args = append(args, p.parseExpression())
	}

	return args
}

// ParseExpression is a convenience function to parse an expression string.
func ParseExpression(input string) (Node, error) {
	parser := NewParser(input)
	return parser.Parse()
}
