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
			name:       "condition with ${{ }} wrapper",
			condition:  "${{ always() }}",
			env:        nil,
			jobSuccess: false,
			want:       true,
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
		// Double negation
		{
			name:       "double negation",
			condition:  "!!true",
			jobSuccess: true,
			want:       true,
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

	// Test hashFiles with directory (should skip directories)
	tmpDir := t.TempDir()
	subDir := tmpDir + "/subdir"
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	// Create a file too
	if err := os.WriteFile(tmpDir+"/file.txt", []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	result, err = evaluator.EvaluateWithWorkDir(
		"hashFiles('*') != ''",
		nil,
		nil,
		true,
		tmpDir,
	)
	if err != nil {
		t.Errorf("hashFiles() with directory unexpected error: %v", err)
	}
	if !result {
		t.Error("hashFiles('*') should return non-empty when files exist")
	}

	// Test hashFiles with invalid glob pattern (should continue without error)
	result, err = evaluator.EvaluateWithWorkDir(
		"hashFiles('[invalid') == ''",
		nil,
		nil,
		true,
		tmpDir,
	)
	if err != nil {
		t.Errorf("hashFiles() with invalid pattern unexpected error: %v", err)
	}
	if !result {
		t.Error("hashFiles('[invalid') should return empty string for invalid pattern")
	}
}

func TestEvaluatorErrorCases(t *testing.T) {
	evaluator := NewConditionEvaluator()

	// Test evaluation error - accessing undefined variable in strict context
	// Note: With AllowUndefinedVariables(), most undefined vars return nil
	// We need to trigger an actual runtime error

	// Test with nil env and accessing nested property that doesn't exist
	// This should still work due to AllowUndefinedVariables
	result, err := evaluator.Evaluate(
		"undefined_var.nested == 'value'",
		nil,
		nil,
		true,
	)
	// The expression should handle undefined gracefully
	if err != nil {
		// If there's an error, result should be true (default)
		if !result {
			t.Errorf("Expected result to be true when error occurs, got false")
		}
	}
}
