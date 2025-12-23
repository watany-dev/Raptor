package expression

import (
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
			name:       "unsupported condition syntax",
			condition:  "some.unsupported.syntax",
			env:        nil,
			jobSuccess: true,
			want:       true,
			wantErr:    true,
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
		t.Error("NewConditionEvaluator() returned nil")
	}
	if evaluator.envCompareRegex == nil {
		t.Error("envCompareRegex is nil")
	}
	if evaluator.envNotEqualRegex == nil {
		t.Error("envNotEqualRegex is nil")
	}
	if evaluator.stepsOutcomeRegex == nil {
		t.Error("stepsOutcomeRegex is nil")
	}
}
