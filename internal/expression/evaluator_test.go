package expression

import (
	"os"
	"testing"
)

func TestConditionEvaluator_Evaluate(t *testing.T) {
	evaluator := NewConditionEvaluator()

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
