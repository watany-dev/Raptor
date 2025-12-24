package expression

import (
	"testing"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		check   func(t *testing.T, node Node)
		wantErr bool
	}{
		{
			name:  "true literal",
			input: "true",
			check: func(t *testing.T, node Node) {
				lit, ok := node.(*BoolLiteral)
				if !ok {
					t.Fatalf("expected BoolLiteral, got %T", node)
				}
				if lit.Value != true {
					t.Errorf("expected true, got %v", lit.Value)
				}
			},
		},
		{
			name:  "false literal",
			input: "false",
			check: func(t *testing.T, node Node) {
				lit, ok := node.(*BoolLiteral)
				if !ok {
					t.Fatalf("expected BoolLiteral, got %T", node)
				}
				if lit.Value != false {
					t.Errorf("expected false, got %v", lit.Value)
				}
			},
		},
		{
			name:  "string literal",
			input: "'hello'",
			check: func(t *testing.T, node Node) {
				lit, ok := node.(*StringLiteral)
				if !ok {
					t.Fatalf("expected StringLiteral, got %T", node)
				}
				if lit.Value != "hello" {
					t.Errorf("expected 'hello', got %q", lit.Value)
				}
			},
		},
		{
			name:  "identifier",
			input: "env.MY_VAR",
			check: func(t *testing.T, node Node) {
				ident, ok := node.(*Identifier)
				if !ok {
					t.Fatalf("expected Identifier, got %T", node)
				}
				if ident.Value != "env.MY_VAR" {
					t.Errorf("expected 'env.MY_VAR', got %q", ident.Value)
				}
			},
		},
		{
			name:  "function call without arguments",
			input: "success()",
			check: func(t *testing.T, node Node) {
				call, ok := node.(*CallExpr)
				if !ok {
					t.Fatalf("expected CallExpr, got %T", node)
				}
				if call.FuncName != "success" {
					t.Errorf("expected 'success', got %q", call.FuncName)
				}
				if len(call.Arguments) != 0 {
					t.Errorf("expected 0 arguments, got %d", len(call.Arguments))
				}
			},
		},
		{
			name:  "function call with arguments",
			input: "contains(github.event.body, '/deploy')",
			check: func(t *testing.T, node Node) {
				call, ok := node.(*CallExpr)
				if !ok {
					t.Fatalf("expected CallExpr, got %T", node)
				}
				if call.FuncName != "contains" {
					t.Errorf("expected 'contains', got %q", call.FuncName)
				}
				if len(call.Arguments) != 2 {
					t.Errorf("expected 2 arguments, got %d", len(call.Arguments))
				}
				// Check first argument
				ident, ok := call.Arguments[0].(*Identifier)
				if !ok {
					t.Fatalf("expected Identifier for arg 0, got %T", call.Arguments[0])
				}
				if ident.Value != "github.event.body" {
					t.Errorf("expected 'github.event.body', got %q", ident.Value)
				}
				// Check second argument
				str, ok := call.Arguments[1].(*StringLiteral)
				if !ok {
					t.Fatalf("expected StringLiteral for arg 1, got %T", call.Arguments[1])
				}
				if str.Value != "/deploy" {
					t.Errorf("expected '/deploy', got %q", str.Value)
				}
			},
		},
		{
			name:  "NOT expression",
			input: "!failure()",
			check: func(t *testing.T, node Node) {
				unary, ok := node.(*UnaryExpr)
				if !ok {
					t.Fatalf("expected UnaryExpr, got %T", node)
				}
				if unary.Operator != TOKEN_NOT {
					t.Errorf("expected TOKEN_NOT, got %v", unary.Operator)
				}
				call, ok := unary.Operand.(*CallExpr)
				if !ok {
					t.Fatalf("expected CallExpr operand, got %T", unary.Operand)
				}
				if call.FuncName != "failure" {
					t.Errorf("expected 'failure', got %q", call.FuncName)
				}
			},
		},
		{
			name:  "AND expression",
			input: "success() && true",
			check: func(t *testing.T, node Node) {
				binary, ok := node.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", node)
				}
				if binary.Operator != TOKEN_AND {
					t.Errorf("expected TOKEN_AND, got %v", binary.Operator)
				}
				// Check left operand
				call, ok := binary.Left.(*CallExpr)
				if !ok {
					t.Fatalf("expected CallExpr for left, got %T", binary.Left)
				}
				if call.FuncName != "success" {
					t.Errorf("expected 'success', got %q", call.FuncName)
				}
				// Check right operand
				lit, ok := binary.Right.(*BoolLiteral)
				if !ok {
					t.Fatalf("expected BoolLiteral for right, got %T", binary.Right)
				}
				if lit.Value != true {
					t.Errorf("expected true, got %v", lit.Value)
				}
			},
		},
		{
			name:  "OR expression",
			input: "success() || failure()",
			check: func(t *testing.T, node Node) {
				binary, ok := node.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", node)
				}
				if binary.Operator != TOKEN_OR {
					t.Errorf("expected TOKEN_OR, got %v", binary.Operator)
				}
			},
		},
		{
			name:  "equality expression",
			input: "env.VAR == 'value'",
			check: func(t *testing.T, node Node) {
				binary, ok := node.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", node)
				}
				if binary.Operator != TOKEN_EQ {
					t.Errorf("expected TOKEN_EQ, got %v", binary.Operator)
				}
			},
		},
		{
			name:  "inequality expression",
			input: "env.VAR != 'value'",
			check: func(t *testing.T, node Node) {
				binary, ok := node.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", node)
				}
				if binary.Operator != TOKEN_NE {
					t.Errorf("expected TOKEN_NE, got %v", binary.Operator)
				}
			},
		},
		{
			name:  "operator precedence: OR is lower than AND",
			input: "a || b && c",
			check: func(t *testing.T, node Node) {
				// Should parse as: a || (b && c)
				binary, ok := node.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", node)
				}
				if binary.Operator != TOKEN_OR {
					t.Errorf("top level should be OR, got %v", binary.Operator)
				}
				// Left should be identifier 'a'
				left, ok := binary.Left.(*Identifier)
				if !ok {
					t.Fatalf("expected Identifier for left, got %T", binary.Left)
				}
				if left.Value != "a" {
					t.Errorf("expected 'a', got %q", left.Value)
				}
				// Right should be AND expression
				right, ok := binary.Right.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr for right, got %T", binary.Right)
				}
				if right.Operator != TOKEN_AND {
					t.Errorf("right should be AND, got %v", right.Operator)
				}
			},
		},
		{
			name:  "grouped expression",
			input: "(success() || failure()) && true",
			check: func(t *testing.T, node Node) {
				// Should parse as: (success() || failure()) && true
				binary, ok := node.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", node)
				}
				if binary.Operator != TOKEN_AND {
					t.Errorf("top level should be AND, got %v", binary.Operator)
				}
				// Left should be OR expression
				left, ok := binary.Left.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr for left, got %T", binary.Left)
				}
				if left.Operator != TOKEN_OR {
					t.Errorf("left should be OR, got %v", left.Operator)
				}
			},
		},
		{
			name:  "double NOT",
			input: "!!true",
			check: func(t *testing.T, node Node) {
				unary, ok := node.(*UnaryExpr)
				if !ok {
					t.Fatalf("expected UnaryExpr, got %T", node)
				}
				if unary.Operator != TOKEN_NOT {
					t.Errorf("expected TOKEN_NOT, got %v", unary.Operator)
				}
				inner, ok := unary.Operand.(*UnaryExpr)
				if !ok {
					t.Fatalf("expected UnaryExpr inside, got %T", unary.Operand)
				}
				if inner.Operator != TOKEN_NOT {
					t.Errorf("expected TOKEN_NOT, got %v", inner.Operator)
				}
			},
		},
		{
			name:  "complex expression",
			input: "success() && env.VAR == 'prod' || failure()",
			check: func(t *testing.T, node Node) {
				// Should parse as: (success() && (env.VAR == 'prod')) || failure()
				binary, ok := node.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", node)
				}
				if binary.Operator != TOKEN_OR {
					t.Errorf("top level should be OR, got %v", binary.Operator)
				}
			},
		},
		{
			name:  "steps outcome",
			input: "steps.build.outcome == 'success'",
			check: func(t *testing.T, node Node) {
				binary, ok := node.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", node)
				}
				ident, ok := binary.Left.(*Identifier)
				if !ok {
					t.Fatalf("expected Identifier for left, got %T", binary.Left)
				}
				if ident.Value != "steps.build.outcome" {
					t.Errorf("expected 'steps.build.outcome', got %q", ident.Value)
				}
			},
		},
		{
			name:  "hashFiles function",
			input: "hashFiles('**/go.sum') != ''",
			check: func(t *testing.T, node Node) {
				binary, ok := node.(*BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", node)
				}
				if binary.Operator != TOKEN_NE {
					t.Errorf("expected TOKEN_NE, got %v", binary.Operator)
				}
				call, ok := binary.Left.(*CallExpr)
				if !ok {
					t.Fatalf("expected CallExpr for left, got %T", binary.Left)
				}
				if call.FuncName != "hashFiles" {
					t.Errorf("expected 'hashFiles', got %q", call.FuncName)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseExpression(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if node == nil {
				t.Fatal("parsed node is nil")
			}
			if tt.check != nil {
				tt.check(t, node)
			}
		})
	}
}

func TestParseExpression(t *testing.T) {
	// Basic sanity check
	node, err := ParseExpression("true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node == nil {
		t.Fatal("node is nil")
	}
}

func TestNewParser(t *testing.T) {
	parser := NewParser("test")
	if parser == nil {
		t.Fatal("NewParser returned nil")
	}
}
