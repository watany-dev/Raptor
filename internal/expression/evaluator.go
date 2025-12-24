package expression

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/expr-lang/expr"
)

// StepContext holds the result of a step for condition evaluation.
type StepContext struct {
	Outcome    string            // "success", "failure", or "skipped"
	Conclusion string            // "success", "failure", or "skipped"
	Outputs    map[string]string // step outputs
}

// EvaluationContext holds all context needed to evaluate an expression.
type EvaluationContext struct {
	Env        map[string]interface{}
	Steps      map[string]interface{}
	JobSuccess bool
	WorkDir    string
}

// ConditionEvaluator evaluates step if conditions.
type ConditionEvaluator struct{}

// NewConditionEvaluator creates a new ConditionEvaluator.
func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{}
}

// Evaluate evaluates the if condition for a step.
// Returns true if the step should run, false if it should be skipped.
func (ce *ConditionEvaluator) Evaluate(
	condition string,
	env map[string]string,
	stepsContext map[string]*StepContext,
	jobSuccess bool,
) (bool, error) {
	return ce.EvaluateWithWorkDir(condition, env, stepsContext, jobSuccess, "")
}

// EvaluateWithWorkDir evaluates the if condition with a working directory for hashFiles().
func (ce *ConditionEvaluator) EvaluateWithWorkDir(
	condition string,
	env map[string]string,
	stepsContext map[string]*StepContext,
	jobSuccess bool,
	workDir string,
) (bool, error) {
	if condition == "" {
		return jobSuccess, nil
	}

	// Normalize condition: remove ${{ }} wrapper if present
	cond := strings.TrimSpace(condition)
	if strings.HasPrefix(cond, "${{") && strings.HasSuffix(cond, "}}") {
		cond = strings.TrimSpace(cond[3 : len(cond)-2])
	}

	// Preprocess: rename functions that conflict with expr built-in operators
	// The expr library uses 'contains', 'startsWith', 'endsWith' as infix operators,
	// but GitHub Actions uses function call syntax. We rename to avoid conflicts.
	cond = preprocessCondition(cond)

	// Build env map for expr
	envMap := make(map[string]interface{})
	for k, v := range env {
		envMap[k] = v
	}

	// Build steps map for expr
	stepsMap := make(map[string]interface{})
	for k, v := range stepsContext {
		stepMap := map[string]interface{}{
			"outcome":    v.Outcome,
			"conclusion": v.Conclusion,
		}
		if v.Outputs != nil {
			outputsMap := make(map[string]interface{})
			for ok, ov := range v.Outputs {
				outputsMap[ok] = ov
			}
			stepMap["outputs"] = outputsMap
		} else {
			stepMap["outputs"] = map[string]interface{}{}
		}
		stepsMap[k] = stepMap
	}

	// Set up working directory for hashFiles
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	// Create evaluation environment
	evalEnv := map[string]interface{}{
		"env":   envMap,
		"steps": stepsMap,
		// Status functions
		"success":   func() bool { return jobSuccess },
		"failure":   func() bool { return !jobSuccess },
		"always":    func() bool { return true },
		"cancelled": func() bool { return false },
	}

	// Define custom functions using expr.Function()
	// Note: We use prefixed names (gha_*) to avoid conflicts with expr built-in operators
	containsFunc := expr.Function(
		"gha_contains",
		func(params ...any) (any, error) {
			s := fmt.Sprintf("%v", params[0])
			substr := fmt.Sprintf("%v", params[1])
			return strings.Contains(strings.ToLower(s), strings.ToLower(substr)), nil
		},
		new(func(string, string) bool),
	)

	startsWithFunc := expr.Function(
		"gha_startsWith",
		func(params ...any) (any, error) {
			s := fmt.Sprintf("%v", params[0])
			prefix := fmt.Sprintf("%v", params[1])
			return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)), nil
		},
		new(func(string, string) bool),
	)

	endsWithFunc := expr.Function(
		"gha_endsWith",
		func(params ...any) (any, error) {
			s := fmt.Sprintf("%v", params[0])
			suffix := fmt.Sprintf("%v", params[1])
			return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix)), nil
		},
		new(func(string, string) bool),
	)

	hashFilesFunc := expr.Function(
		"gha_hashFiles",
		func(params ...any) (any, error) {
			patterns := make([]string, len(params))
			for i, p := range params {
				patterns[i] = fmt.Sprintf("%v", p)
			}
			return hashFiles(patterns, workDir), nil
		},
		new(func(...string) string),
	)

	// Compile and run the expression
	program, err := expr.Compile(cond,
		expr.Env(evalEnv),
		expr.AsBool(),
		containsFunc,
		startsWithFunc,
		endsWithFunc,
		hashFilesFunc,
		expr.AllowUndefinedVariables(),
	)
	if err != nil {
		return true, fmt.Errorf("parse error: %v (defaulting to true)", err)
	}

	result, err := expr.Run(program, evalEnv)
	if err != nil {
		return true, fmt.Errorf("evaluation error: %v (defaulting to true)", err)
	}

	if b, ok := result.(bool); ok {
		return b, nil
	}

	return true, fmt.Errorf("expression did not return boolean (defaulting to true)")
}

// preprocessCondition converts GitHub Actions function names to expr-compatible names.
// This is needed because 'contains', 'startsWith', and 'endsWith' are built-in operators
// in the expr library, but GitHub Actions uses them as function calls.
func preprocessCondition(cond string) string {
	// Replace function names with prefixed versions to avoid operator conflicts
	// We need to be careful to only replace function calls, not arbitrary text
	replacements := []struct {
		from string
		to   string
	}{
		{"contains(", "gha_contains("},
		{"startsWith(", "gha_startsWith("},
		{"endsWith(", "gha_endsWith("},
		{"hashFiles(", "gha_hashFiles("},
	}

	result := cond
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r.from, r.to)
	}
	return result
}

// hashFiles calculates SHA256 hash of files matching the patterns.
func hashFiles(patterns []string, workDir string) string {
	if len(patterns) == 0 {
		return ""
	}

	var allBytes []byte
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(workDir, pattern))
		if err != nil {
			continue
		}

		sort.Strings(matches)

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			allBytes = append(allBytes, data...)
		}
	}

	if len(allBytes) == 0 {
		return ""
	}

	hash := sha256.Sum256(allBytes)
	return hex.EncodeToString(hash[:])
}
