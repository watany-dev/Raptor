package expression

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// StepContext holds the result of a step for condition evaluation.
type StepContext struct {
	Outcome    string            // "success", "failure", or "skipped"
	Conclusion string            // "success", "failure", or "skipped"
	Outputs    map[string]string // step outputs
}

// EvaluationContext holds all context needed to evaluate an expression.
type EvaluationContext struct {
	Env          map[string]string
	StepsContext map[string]*StepContext
	JobSuccess   bool
	WorkDir      string // for hashFiles()
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
	// If no condition is specified, use default behavior (run if previous steps succeeded)
	if condition == "" {
		return jobSuccess, nil
	}

	// Normalize condition: remove ${{ }} wrapper if present
	cond := strings.TrimSpace(condition)
	if strings.HasPrefix(cond, "${{") && strings.HasSuffix(cond, "}}") {
		cond = strings.TrimSpace(cond[3 : len(cond)-2])
	}

	// Parse the expression
	node, err := ParseExpression(cond)
	if err != nil {
		return true, fmt.Errorf("parse error: %v (defaulting to true)", err)
	}

	// Create evaluation context
	ctx := &EvaluationContext{
		Env:          env,
		StepsContext: stepsContext,
		JobSuccess:   jobSuccess,
		WorkDir:      workDir,
	}

	// Evaluate the AST
	result, err := evaluateNode(node, ctx)
	if err != nil {
		return true, fmt.Errorf("evaluation error: %v (defaulting to true)", err)
	}

	// Convert result to boolean
	return toBool(result), nil
}

// evaluateNode evaluates an AST node and returns the result.
func evaluateNode(node Node, ctx *EvaluationContext) (interface{}, error) {
	switch n := node.(type) {
	case *BoolLiteral:
		return n.Value, nil

	case *StringLiteral:
		return n.Value, nil

	case *Identifier:
		return resolveIdentifier(n.Value, ctx)

	case *CallExpr:
		return evaluateCall(n, ctx)

	case *UnaryExpr:
		return evaluateUnary(n, ctx)

	case *BinaryExpr:
		return evaluateBinary(n, ctx)

	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

// resolveIdentifier resolves an identifier to its value.
func resolveIdentifier(name string, ctx *EvaluationContext) (interface{}, error) {
	// Handle env.VAR
	if strings.HasPrefix(name, "env.") {
		varName := strings.TrimPrefix(name, "env.")
		if ctx.Env != nil {
			return ctx.Env[varName], nil
		}
		return "", nil
	}

	// Handle steps.ID.outcome or steps.ID.conclusion
	if strings.HasPrefix(name, "steps.") {
		parts := strings.SplitN(name, ".", 3)
		if len(parts) >= 3 && ctx.StepsContext != nil {
			stepID := parts[1]
			property := parts[2]
			if step, ok := ctx.StepsContext[stepID]; ok {
				switch property {
				case "outcome":
					return step.Outcome, nil
				case "conclusion":
					return step.Conclusion, nil
				}
				// Handle steps.ID.outputs.NAME
				if strings.HasPrefix(property, "outputs.") {
					outputName := strings.TrimPrefix(property, "outputs.")
					if step.Outputs != nil {
						return step.Outputs[outputName], nil
					}
				}
			}
		}
		return "", nil
	}

	// Return the identifier as-is (could be a variable reference we don't understand)
	return name, nil
}

// evaluateCall evaluates a function call expression.
func evaluateCall(call *CallExpr, ctx *EvaluationContext) (interface{}, error) {
	switch call.FuncName {
	case "always":
		return true, nil

	case "success":
		return ctx.JobSuccess, nil

	case "failure":
		return !ctx.JobSuccess, nil

	case "cancelled":
		return false, nil

	case "contains":
		return evalContains(call.Arguments, ctx)

	case "startsWith":
		return evalStartsWith(call.Arguments, ctx)

	case "endsWith":
		return evalEndsWith(call.Arguments, ctx)

	case "hashFiles":
		return evalHashFiles(call.Arguments, ctx)

	default:
		return nil, fmt.Errorf("unknown function: %s", call.FuncName)
	}
}

// evaluateUnary evaluates a unary expression.
func evaluateUnary(unary *UnaryExpr, ctx *EvaluationContext) (interface{}, error) {
	operand, err := evaluateNode(unary.Operand, ctx)
	if err != nil {
		return nil, err
	}

	switch unary.Operator {
	case TOKEN_NOT:
		return !toBool(operand), nil
	default:
		return nil, fmt.Errorf("unknown unary operator: %v", unary.Operator)
	}
}

// evaluateBinary evaluates a binary expression.
func evaluateBinary(binary *BinaryExpr, ctx *EvaluationContext) (interface{}, error) {
	// Short-circuit evaluation for && and ||
	switch binary.Operator {
	case TOKEN_AND:
		left, err := evaluateNode(binary.Left, ctx)
		if err != nil {
			return nil, err
		}
		if !toBool(left) {
			return false, nil // Short-circuit: false && anything = false
		}
		right, err := evaluateNode(binary.Right, ctx)
		if err != nil {
			return nil, err
		}
		return toBool(right), nil

	case TOKEN_OR:
		left, err := evaluateNode(binary.Left, ctx)
		if err != nil {
			return nil, err
		}
		if toBool(left) {
			return true, nil // Short-circuit: true || anything = true
		}
		right, err := evaluateNode(binary.Right, ctx)
		if err != nil {
			return nil, err
		}
		return toBool(right), nil

	case TOKEN_EQ:
		left, err := evaluateNode(binary.Left, ctx)
		if err != nil {
			return nil, err
		}
		right, err := evaluateNode(binary.Right, ctx)
		if err != nil {
			return nil, err
		}
		return toString(left) == toString(right), nil

	case TOKEN_NE:
		left, err := evaluateNode(binary.Left, ctx)
		if err != nil {
			return nil, err
		}
		right, err := evaluateNode(binary.Right, ctx)
		if err != nil {
			return nil, err
		}
		return toString(left) != toString(right), nil

	default:
		return nil, fmt.Errorf("unknown binary operator: %v", binary.Operator)
	}
}

// evalContains evaluates the contains(haystack, needle) function.
// Returns true if haystack contains needle (case-insensitive).
func evalContains(args []Node, ctx *EvaluationContext) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("contains() requires 2 arguments, got %d", len(args))
	}

	haystack, err := evaluateNode(args[0], ctx)
	if err != nil {
		return false, err
	}
	needle, err := evaluateNode(args[1], ctx)
	if err != nil {
		return false, err
	}

	return strings.Contains(
		strings.ToLower(toString(haystack)),
		strings.ToLower(toString(needle)),
	), nil
}

// evalStartsWith evaluates the startsWith(str, prefix) function.
// Returns true if str starts with prefix (case-insensitive).
func evalStartsWith(args []Node, ctx *EvaluationContext) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("startsWith() requires 2 arguments, got %d", len(args))
	}

	str, err := evaluateNode(args[0], ctx)
	if err != nil {
		return false, err
	}
	prefix, err := evaluateNode(args[1], ctx)
	if err != nil {
		return false, err
	}

	return strings.HasPrefix(
		strings.ToLower(toString(str)),
		strings.ToLower(toString(prefix)),
	), nil
}

// evalEndsWith evaluates the endsWith(str, suffix) function.
// Returns true if str ends with suffix (case-insensitive).
func evalEndsWith(args []Node, ctx *EvaluationContext) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("endsWith() requires 2 arguments, got %d", len(args))
	}

	str, err := evaluateNode(args[0], ctx)
	if err != nil {
		return false, err
	}
	suffix, err := evaluateNode(args[1], ctx)
	if err != nil {
		return false, err
	}

	return strings.HasSuffix(
		strings.ToLower(toString(str)),
		strings.ToLower(toString(suffix)),
	), nil
}

// evalHashFiles evaluates the hashFiles(pattern...) function.
// Returns the SHA256 hash of the contents of files matching the patterns.
func evalHashFiles(args []Node, ctx *EvaluationContext) (string, error) {
	if len(args) == 0 {
		return "", nil
	}

	workDir := ctx.WorkDir
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return "", nil
		}
	}

	var allBytes []byte
	for _, arg := range args {
		patternVal, err := evaluateNode(arg, ctx)
		if err != nil {
			return "", err
		}
		pattern := toString(patternVal)

		// Use doublestar for ** glob patterns
		matches, err := filepath.Glob(filepath.Join(workDir, pattern))
		if err != nil {
			continue
		}

		// Sort for consistent hashing
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
		return "", nil
	}

	hash := sha256.Sum256(allBytes)
	return hex.EncodeToString(hash[:]), nil
}

// toBool converts a value to a boolean.
func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "false" && val != "0"
	case int:
		return val != 0
	case float64:
		return val != 0
	default:
		return v != nil
	}
}

// toString converts a value to a string.
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
