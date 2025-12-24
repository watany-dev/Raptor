package expression

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkEvalContains benchmarks the contains() function evaluation.
func BenchmarkEvalContains(b *testing.B) {
	evaluator := NewConditionEvaluator()
	env := map[string]string{
		"GITHUB_REF": "refs/heads/feature/my-awesome-feature-branch",
		"RUNNER_OS":  "Linux",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.Evaluate("contains(env.GITHUB_REF, 'feature')", env, nil, true)
	}
}

// BenchmarkEvalStartsWith benchmarks the startsWith() function evaluation.
func BenchmarkEvalStartsWith(b *testing.B) {
	evaluator := NewConditionEvaluator()
	env := map[string]string{
		"GITHUB_REF": "refs/heads/feature/my-awesome-feature-branch",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.Evaluate("startsWith(env.GITHUB_REF, 'refs/heads/')", env, nil, true)
	}
}

// BenchmarkEvalEndsWith benchmarks the endsWith() function evaluation.
func BenchmarkEvalEndsWith(b *testing.B) {
	evaluator := NewConditionEvaluator()
	env := map[string]string{
		"FILENAME": "my-document.pdf",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.Evaluate("endsWith(env.FILENAME, '.pdf')", env, nil, true)
	}
}

// BenchmarkEvalContainsLongString benchmarks contains() with longer strings.
func BenchmarkEvalContainsLongString(b *testing.B) {
	evaluator := NewConditionEvaluator()
	// Simulate a long environment variable value
	longValue := "This is a very long string that simulates a complex environment variable value with many characters and multiple words to test the performance of the contains function with larger inputs"
	env := map[string]string{
		"LONG_VAR": longValue,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.Evaluate("contains(env.LONG_VAR, 'performance')", env, nil, true)
	}
}

// BenchmarkEvalComplexCondition benchmarks a complex condition with multiple operators.
func BenchmarkEvalComplexCondition(b *testing.B) {
	evaluator := NewConditionEvaluator()
	env := map[string]string{
		"GITHUB_REF":   "refs/heads/main",
		"GITHUB_ACTOR": "developer",
		"RUNNER_OS":    "Linux",
		"GITHUB_EVENT": "push",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.Evaluate(
			"(contains(env.GITHUB_REF, 'main') || startsWith(env.GITHUB_REF, 'refs/tags/')) && env.RUNNER_OS == 'Linux'",
			env, nil, true)
	}
}

// BenchmarkEvalSuccess benchmarks the success() function.
func BenchmarkEvalSuccess(b *testing.B) {
	evaluator := NewConditionEvaluator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.Evaluate("success()", nil, nil, true)
	}
}

// BenchmarkEvalAlways benchmarks the always() function.
func BenchmarkEvalAlways(b *testing.B) {
	evaluator := NewConditionEvaluator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.Evaluate("always()", nil, nil, true)
	}
}

// BenchmarkEvalStepsContext benchmarks step context access.
func BenchmarkEvalStepsContext(b *testing.B) {
	evaluator := NewConditionEvaluator()
	stepsCtx := map[string]*StepContext{
		"build": {Outcome: "success", Conclusion: "success"},
		"test":  {Outcome: "success", Conclusion: "success"},
		"lint":  {Outcome: "failure", Conclusion: "failure"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.Evaluate("steps.build.outcome == 'success' && steps.test.outcome == 'success'", nil, stepsCtx, true)
	}
}

// BenchmarkParseExpression benchmarks just the parsing phase.
func BenchmarkParseExpression(b *testing.B) {
	expr := "contains(env.GITHUB_REF, 'feature') && startsWith(env.GITHUB_ACTOR, 'dev')"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseExpression(expr)
	}
}

// BenchmarkParseComplexExpression benchmarks parsing a complex expression.
func BenchmarkParseComplexExpression(b *testing.B) {
	expr := "(contains(env.GITHUB_REF, 'main') || startsWith(env.GITHUB_REF, 'refs/tags/')) && env.RUNNER_OS == 'Linux' && !failure()"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseExpression(expr)
	}
}

// BenchmarkTokenize benchmarks the tokenization phase.
func BenchmarkTokenize(b *testing.B) {
	expr := "contains(env.GITHUB_REF, 'feature') && startsWith(env.GITHUB_ACTOR, 'dev')"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Tokenize(expr)
	}
}

// BenchmarkHashFilesSmall benchmarks hashFiles with a small file.
func BenchmarkHashFilesSmall(b *testing.B) {
	// Create a temporary directory with a small test file
	tmpDir, err := os.MkdirTemp("", "benchmark-hashfiles-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a small test file (1KB)
	testFile := filepath.Join(tmpDir, "small.txt")
	content := make([]byte, 1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		b.Fatal(err)
	}

	evaluator := NewConditionEvaluator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.EvaluateWithWorkDir("hashFiles('small.txt')", nil, nil, true, tmpDir)
	}
}

// BenchmarkHashFilesMedium benchmarks hashFiles with a medium file.
func BenchmarkHashFilesMedium(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "benchmark-hashfiles-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a medium test file (100KB)
	testFile := filepath.Join(tmpDir, "medium.txt")
	content := make([]byte, 100*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		b.Fatal(err)
	}

	evaluator := NewConditionEvaluator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.EvaluateWithWorkDir("hashFiles('medium.txt')", nil, nil, true, tmpDir)
	}
}

// BenchmarkHashFilesMultiple benchmarks hashFiles with multiple files.
func BenchmarkHashFilesMultiple(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "benchmark-hashfiles-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple test files
	for j := 0; j < 10; j++ {
		testFile := filepath.Join(tmpDir, "file"+string(rune('0'+j))+".txt")
		content := make([]byte, 10*1024) // 10KB each
		for i := range content {
			content[i] = byte((i + j) % 256)
		}
		if err := os.WriteFile(testFile, content, 0644); err != nil {
			b.Fatal(err)
		}
	}

	evaluator := NewConditionEvaluator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.EvaluateWithWorkDir("hashFiles('*.txt')", nil, nil, true, tmpDir)
	}
}

// BenchmarkRepeatedEvaluation simulates evaluating the same condition multiple times
// (like in a workflow with many steps using the same condition)
func BenchmarkRepeatedEvaluation(b *testing.B) {
	evaluator := NewConditionEvaluator()
	env := map[string]string{
		"GITHUB_REF": "refs/heads/main",
	}
	condition := "success()"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate 10 steps with the same condition
		for j := 0; j < 10; j++ {
			evaluator.Evaluate(condition, env, nil, true)
		}
	}
}

// BenchmarkStringFunctionComparison compares the three string functions.
func BenchmarkStringFunctionComparison(b *testing.B) {
	evaluator := NewConditionEvaluator()
	env := map[string]string{
		"VAR": "Hello World Example String",
	}

	b.Run("contains", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evaluator.Evaluate("contains(env.VAR, 'World')", env, nil, true)
		}
	})

	b.Run("startsWith", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evaluator.Evaluate("startsWith(env.VAR, 'Hello')", env, nil, true)
		}
	})

	b.Run("endsWith", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evaluator.Evaluate("endsWith(env.VAR, 'String')", env, nil, true)
		}
	})
}
