package expression

import (
	"os"
	"testing"
)

func TestConditionEvaluator_Evaluate(t *testing.T) {
	evaluator := NewConditionEvaluator()
	evaluator.StrictMode = false // Test permissive mode for backward compatibility

	tests := []struct {
		name         string
		condition    string
		env          map[string]string
		stepsContext map[string]*StepContext
		jobSuccess   bool
		want         bool
		wantErr      bool
	}{
		{
			name:       "empty condition with job success",
			condition:  "",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "empty condition with job failure",
			condition:  "",
			env:        nil,
			jobSuccess: false,
			want:       false,
		},
		{
			name:       "true literal",
			condition:  "true",
			env:        nil,
			jobSuccess: false,
			want:       true,
		},
		{
			name:       "false literal",
			condition:  "false",
			env:        nil,
			jobSuccess: true,
			want:       false,
		},
		{
			name:       "always() function",
			condition:  "always()",
			env:        nil,
			jobSuccess: false,
			want:       true,
		},
		{
			name:       "success() function with job success",
			condition:  "success()",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "success() function with job failure",
			condition:  "success()",
			env:        nil,
			jobSuccess: false,
			want:       false,
		},
		{
			name:       "failure() function with job success",
			condition:  "failure()",
			env:        nil,
			jobSuccess: true,
			want:       false,
		},
		{
			name:       "failure() function with job failure",
			condition:  "failure()",
			env:        nil,
			jobSuccess: false,
			want:       true,
		},
		{
			name:       "cancelled() function",
			condition:  "cancelled()",
			env:        nil,
			jobSuccess: true,
			want:       false,
		},
		{
			name:       "env comparison equals - match",
			condition:  "env.MY_VAR == 'expected'",
			env:        map[string]string{"MY_VAR": "expected"},
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "env comparison equals - no match",
			condition:  "env.MY_VAR == 'expected'",
			env:        map[string]string{"MY_VAR": "different"},
			jobSuccess: true,
			want:       false,
		},
		{
			name:       "env comparison not equals - match",
			condition:  "env.MY_VAR != 'other'",
			env:        map[string]string{"MY_VAR": "value"},
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "env comparison not equals - no match",
			condition:  "env.MY_VAR != 'value'",
			env:        map[string]string{"MY_VAR": "value"},
			jobSuccess: true,
			want:       false,
		},
		{
			name:      "steps outcome comparison - match",
			condition: "steps.build.outcome == 'success'",
			env:       nil,
			stepsContext: map[string]*StepContext{
				"build": {Outcome: "success", Conclusion: "success"},
			},
			jobSuccess: true,
			want:       true,
		},
		{
			name:      "steps outcome comparison - no match",
			condition: "steps.build.outcome == 'success'",
			env:       nil,
			stepsContext: map[string]*StepContext{
				"build": {Outcome: "failure", Conclusion: "failure"},
			},
			jobSuccess: true,
			want:       false,
		},
		{
			name:         "steps outcome comparison - step not found",
			condition:    "steps.nonexistent.outcome == 'success'",
			env:          nil,
			stepsContext: map[string]*StepContext{},
			jobSuccess:   true,
			want:         false,
		},
		{
			name:       "condition with ${{ }} wrapper",
			condition:  "${{ always() }}",
			env:        nil,
			jobSuccess: false,
			want:       true,
		},
		{
			name:       "identifier resolves to truthy value",
			condition:  "some.unsupported.syntax",
			env:        nil,
			jobSuccess: true,
			want:       true,
			wantErr:    false,
		},
		{
			name:       "parse error returns true with error",
			condition:  "@invalid",
			env:        nil,
			jobSuccess: true,
			want:       true,
			wantErr:    true,
		},
		// Logical operators tests
		{
			name:       "AND operator - both true",
			condition:  "success() && true",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "AND operator - left false",
			condition:  "success() && true",
			env:        nil,
			jobSuccess: false,
			want:       false,
		},
		{
			name:       "AND operator - right false",
			condition:  "success() && failure()",
			env:        nil,
			jobSuccess: true,
			want:       false,
		},
		{
			name:       "OR operator - both true",
			condition:  "success() || true",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "OR operator - left true",
			condition:  "success() || failure()",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "OR operator - right true",
			condition:  "failure() || true",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "OR operator - both false",
			condition:  "failure() || false",
			env:        nil,
			jobSuccess: true,
			want:       false,
		},
		{
			name:       "NOT operator - negate true",
			condition:  "!failure()",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "NOT operator - negate false",
			condition:  "!success()",
			env:        nil,
			jobSuccess: true,
			want:       false,
		},
		{
			name:       "complex expression with AND and NOT",
			condition:  "success() && !failure()",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "grouped expression",
			condition:  "(success() || failure()) && true",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "mixed operators with env",
			condition:  "success() && env.MY_VAR == 'prod'",
			env:        map[string]string{"MY_VAR": "prod"},
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "mixed operators with env - no match",
			condition:  "success() && env.MY_VAR == 'prod'",
			env:        map[string]string{"MY_VAR": "dev"},
			jobSuccess: true,
			want:       false,
		},
		// contains() function tests
		{
			name:       "contains - match",
			condition:  "contains('Hello World', 'world')",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "contains - no match",
			condition:  "contains('Hello', 'xyz')",
			env:        nil,
			jobSuccess: true,
			want:       false,
		},
		{
			name:       "contains with env variable",
			condition:  "contains(env.MESSAGE, 'deploy')",
			env:        map[string]string{"MESSAGE": "please deploy now"},
			jobSuccess: true,
			want:       true,
		},
		// startsWith() function tests
		{
			name:       "startsWith - match",
			condition:  "startsWith('refs/tags/v1', 'refs/tags/')",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "startsWith - case insensitive match",
			condition:  "startsWith('Hello', 'he')",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "startsWith - no match",
			condition:  "startsWith('Hello', 'World')",
			env:        nil,
			jobSuccess: true,
			want:       false,
		},
		// endsWith() function tests
		{
			name:       "endsWith - match",
			condition:  "endsWith('hello.txt', '.txt')",
			env:        nil,
			jobSuccess: true,
			want:       true,
		},
		{
			name:       "endsWith - no match",
			condition:  "endsWith('hello.txt', '.md')",
			env:        nil,
			jobSuccess: true,
			want:       false,
		},
		// Short-circuit evaluation tests
		{
			name:       "short-circuit AND - left false",
			condition:  "false && unknown_func()",
			env:        nil,
			jobSuccess: true,
			want:       false,
			wantErr:    false, // Should not error due to short-circuit
		},
		{
			name:       "short-circuit OR - left true",
			condition:  "true || unknown_func()",
			env:        nil,
			jobSuccess: true,
			want:       true,
			wantErr:    false, // Should not error due to short-circuit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepsContext := tt.stepsContext
			if stepsContext == nil {
				stepsContext = make(map[string]*StepContext)
			}

			got, err := evaluator.Evaluate(tt.condition, tt.env, stepsContext, tt.jobSuccess)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewConditionEvaluator(t *testing.T) {
	evaluator := NewConditionEvaluator()
	if evaluator == nil {
		t.Fatal("NewConditionEvaluator() returned nil")
	}
}

func TestHashFiles(t *testing.T) {
	evaluator := NewConditionEvaluator()

	// Create a temporary directory with a test file
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test with existing file
	result, err := evaluator.EvaluateWithWorkDir(
		"hashFiles('test.txt') != ''",
		nil,
		nil,
		true,
		tmpDir,
	)
	if err != nil {
		t.Errorf("hashFiles() unexpected error: %v", err)
	}
	if !result {
		t.Error("hashFiles('test.txt') should return non-empty hash")
	}

	// Test with non-existing file
	result, err = evaluator.EvaluateWithWorkDir(
		"hashFiles('nonexistent.file') != ''",
		nil,
		nil,
		true,
		tmpDir,
	)
	if err != nil {
		t.Errorf("hashFiles() unexpected error: %v", err)
	}
	if result {
		t.Error("hashFiles('nonexistent.file') should return empty string")
	}
}

func TestEvaluator_AdditionalCoverage(t *testing.T) {
	evaluator := NewConditionEvaluator()
	evaluator.StrictMode = false // Test permissive mode

	tests := []struct {
		name         string
		condition    string
		env          map[string]string
		stepsContext map[string]*StepContext
		jobSuccess   bool
		want         bool
		wantErr      bool
	}{
		// steps.ID.conclusion test
		{
			name:      "steps conclusion comparison",
			condition: "steps.build.conclusion == 'success'",
			stepsContext: map[string]*StepContext{
				"build": {Outcome: "success", Conclusion: "success"},
			},
			jobSuccess: true,
			want:       true,
		},
		// steps.ID.outputs.NAME test
		{
			name:      "steps outputs comparison",
			condition: "steps.build.outputs.result == 'passed'",
			stepsContext: map[string]*StepContext{
				"build": {
					Outcome:    "success",
					Conclusion: "success",
					Outputs:    map[string]string{"result": "passed"},
				},
			},
			jobSuccess: true,
			want:       true,
		},
		// Unknown function test
		{
			name:       "unknown function returns error",
			condition:  "unknownFunc()",
			jobSuccess: true,
			want:       true,
			wantErr:    true,
		},
		// Function with wrong number of arguments
		{
			name:       "contains with wrong args",
			condition:  "contains('only one arg')",
			jobSuccess: true,
			want:       true,
			wantErr:    true,
		},
		{
			name:       "startsWith with wrong args",
			condition:  "startsWith('only one arg')",
			jobSuccess: true,
			want:       true,
			wantErr:    true,
		},
		{
			name:       "endsWith with wrong args",
			condition:  "endsWith('only one arg')",
			jobSuccess: true,
			want:       true,
			wantErr:    true,
		},
		// Double negation
		{
			name:       "double negation",
			condition:  "!!true",
			jobSuccess: true,
			want:       true,
		},
		// Empty env variable
		{
			name:       "empty env variable equals empty string",
			condition:  "env.MISSING == ''",
			env:        map[string]string{},
			jobSuccess: true,
			want:       true,
		},
		// Missing step returns empty
		{
			name:         "missing step outcome",
			condition:    "steps.missing.outcome == ''",
			stepsContext: map[string]*StepContext{},
			jobSuccess:   true,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepsContext := tt.stepsContext
			if stepsContext == nil {
				stepsContext = make(map[string]*StepContext)
			}
			env := tt.env
			if env == nil {
				env = make(map[string]string)
			}

			got, err := evaluator.Evaluate(tt.condition, env, stepsContext, tt.jobSuccess)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string non-empty", "hello", true},
		{"string empty", "", false},
		{"string false", "false", false},
		{"string 0", "0", false},
		{"int non-zero", 42, true},
		{"int zero", 0, false},
		{"float non-zero", 3.14, true},
		{"float zero", 0.0, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toBool(tt.input)
			if got != tt.want {
				t.Errorf("toBool(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"string", "hello", "hello"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 42, "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toString(tt.input)
			if got != tt.want {
				t.Errorf("toString(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestASTNodes tests that AST node types implement the Node interface
func TestASTNodes(t *testing.T) {
	// These calls exercise the node() methods for coverage
	var nodes []Node
	nodes = append(nodes, &BinaryExpr{})
	nodes = append(nodes, &UnaryExpr{})
	nodes = append(nodes, &CallExpr{})
	nodes = append(nodes, &Identifier{})
	nodes = append(nodes, &StringLiteral{})
	nodes = append(nodes, &BoolLiteral{})

	// Call node() on each to satisfy interface and coverage
	for _, n := range nodes {
		n.node() // This exercises the empty node() methods
	}

	if len(nodes) != 6 {
		t.Error("Expected 6 node types")
	}
}

func TestParserErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"missing closing paren", "(true", true},
		{"missing closing paren in func", "contains('a', 'b'", true},
		{"unexpected token", "true &&", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseExpression(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseExpression(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestHashFilesEdgeCases(t *testing.T) {
	evaluator := NewConditionEvaluator()

	// Test hashFiles with no arguments returns empty
	result, err := evaluator.EvaluateWithWorkDir(
		"hashFiles() == ''",
		nil,
		nil,
		true,
		t.TempDir(),
	)
	if err != nil {
		t.Errorf("hashFiles() unexpected error: %v", err)
	}
	if !result {
		t.Error("hashFiles() with no args should return empty string")
	}
}

func TestErrorPropagation(t *testing.T) {
	evaluator := NewConditionEvaluator()
	evaluator.StrictMode = false // Test permissive mode

	tests := []struct {
		name      string
		condition string
	}{
		// Errors in binary expressions (non-short-circuit path)
		{"error in AND right side", "true && unknownFunc()"},
		{"error in OR right side", "false || unknownFunc()"},
		{"error in EQ left side", "unknownFunc() == 'value'"},
		{"error in EQ right side", "'value' == unknownFunc()"},
		{"error in NE left side", "unknownFunc() != 'value'"},
		{"error in NE right side", "'value' != unknownFunc()"},
		// Errors in unary expressions
		{"error in NOT operand", "!unknownFunc()"},
		// Errors in function arguments
		{"error in contains first arg", "contains(unknownFunc(), 'x')"},
		{"error in contains second arg", "contains('x', unknownFunc())"},
		{"error in startsWith first arg", "startsWith(unknownFunc(), 'x')"},
		{"error in startsWith second arg", "startsWith('x', unknownFunc())"},
		{"error in endsWith first arg", "endsWith(unknownFunc(), 'x')"},
		{"error in endsWith second arg", "endsWith('x', unknownFunc())"},
		{"error in hashFiles arg", "hashFiles(unknownFunc())"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := evaluator.Evaluate(tt.condition, nil, nil, true)
			if err == nil {
				t.Errorf("expected error for %q", tt.condition)
			}
		})
	}
}

func TestStrictMode(t *testing.T) {
	tests := []struct {
		name       string
		strictMode bool
		condition  string
		wantResult bool
		wantErr    bool
	}{
		{
			name:       "strict mode - parse error returns false with error",
			strictMode: true,
			condition:  "@invalid",
			wantResult: false,
			wantErr:    true,
		},
		{
			name:       "permissive mode - parse error returns true with error",
			strictMode: false,
			condition:  "@invalid",
			wantResult: true,
			wantErr:    true,
		},
		{
			name:       "strict mode - unknown function returns false with error",
			strictMode: true,
			condition:  "unknownFunc()",
			wantResult: false,
			wantErr:    true,
		},
		{
			name:       "permissive mode - unknown function returns true with error",
			strictMode: false,
			condition:  "unknownFunc()",
			wantResult: true,
			wantErr:    true,
		},
		{
			name:       "strict mode - valid condition works normally",
			strictMode: true,
			condition:  "true",
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "permissive mode - valid condition works normally",
			strictMode: false,
			condition:  "true",
			wantResult: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator()
			evaluator.StrictMode = tt.strictMode

			got, err := evaluator.Evaluate(tt.condition, nil, nil, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantResult {
				t.Errorf("Evaluate() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}
