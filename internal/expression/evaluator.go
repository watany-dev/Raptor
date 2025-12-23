package expression

import (
	"fmt"
	"regexp"
	"strings"
)

// StepContext holds the result of a step for condition evaluation.
type StepContext struct {
	Outcome    string            // "success", "failure", or "skipped"
	Conclusion string            // "success", "failure", or "skipped"
	Outputs    map[string]string // step outputs
}

// ConditionEvaluator evaluates step if conditions.
// It pre-compiles regex patterns for better performance.
type ConditionEvaluator struct {
	envCompareRegex   *regexp.Regexp
	envNotEqualRegex  *regexp.Regexp
	stepsOutcomeRegex *regexp.Regexp
}

// NewConditionEvaluator creates a new ConditionEvaluator with pre-compiled regex patterns.
func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{
		envCompareRegex:   regexp.MustCompile(`env\.(\w+)\s*==\s*'([^']*)'`),
		envNotEqualRegex:  regexp.MustCompile(`env\.(\w+)\s*!=\s*'([^']*)'`),
		stepsOutcomeRegex: regexp.MustCompile(`steps\.(\w+)\.outcome\s*==\s*'([^']*)'`),
	}
}

// Evaluate evaluates the if condition for a step.
// Returns true if the step should run, false if it should be skipped.
func (ce *ConditionEvaluator) Evaluate(
	condition string,
	env map[string]string,
	stepsContext map[string]*StepContext,
	jobSuccess bool,
) (bool, error) {
	// If no condition is specified, use default behavior (run if previous steps succeeded)
	if condition == "" {
		return jobSuccess, nil
	}

	// Normalize condition: remove ${{ }} wrapper if present
	cond := strings.TrimSpace(condition)
	if strings.HasPrefix(cond, "${{") && strings.HasSuffix(cond, "}}") {
		cond = strings.TrimSpace(cond[3 : len(cond)-2])
	}

	// Handle boolean literals
	if cond == "true" {
		return true, nil
	}
	if cond == "false" {
		return false, nil
	}

	// Handle status check functions
	if cond == "always()" {
		return true, nil
	}
	if cond == "success()" {
		return jobSuccess, nil
	}
	if cond == "failure()" {
		return !jobSuccess, nil
	}
	if cond == "cancelled()" {
		return false, nil // We don't support cancellation yet
	}

	// Handle env.VAR == 'value' comparisons
	if matches := ce.envCompareRegex.FindStringSubmatch(cond); matches != nil {
		varName := matches[1]
		expectedValue := matches[2]
		actualValue := env[varName]
		return actualValue == expectedValue, nil
	}

	// Handle env.VAR != 'value' comparisons
	if matches := ce.envNotEqualRegex.FindStringSubmatch(cond); matches != nil {
		varName := matches[1]
		expectedValue := matches[2]
		actualValue := env[varName]
		return actualValue != expectedValue, nil
	}

	// Handle steps.ID.outcome == 'value' comparisons
	if matches := ce.stepsOutcomeRegex.FindStringSubmatch(cond); matches != nil {
		stepID := matches[1]
		expectedOutcome := matches[2]
		if step, ok := stepsContext[stepID]; ok {
			return step.Outcome == expectedOutcome, nil
		}
		return false, nil
	}

	// For unsupported expressions, default to running the step (with warning)
	return true, fmt.Errorf("unsupported condition syntax: %s (defaulting to true)", cond)
}
