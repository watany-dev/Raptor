package expression

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// ============================================================
// OPTIMIZED IMPLEMENTATIONS FOR BENCHMARKING
// ============================================================

// Optimized evalContains - reduces string allocations
func evalContainsOptimized(args []Node, ctx *EvaluationContext) (bool, error) {
	if len(args) != 2 {
		return false, nil
	}

	haystack, err := evaluateNode(args[0], ctx)
	if err != nil {
		return false, err
	}
	needle, err := evaluateNode(args[1], ctx)
	if err != nil {
		return false, err
	}

	// Optimization: call toString and ToLower once each, store in variables
	haystackStr := strings.ToLower(toString(haystack))
	needleStr := strings.ToLower(toString(needle))
	return strings.Contains(haystackStr, needleStr), nil
}

// Optimized evalStartsWith - reduces string allocations
func evalStartsWithOptimized(args []Node, ctx *EvaluationContext) (bool, error) {
	if len(args) != 2 {
		return false, nil
	}

	str, err := evaluateNode(args[0], ctx)
	if err != nil {
		return false, err
	}
	prefix, err := evaluateNode(args[1], ctx)
	if err != nil {
		return false, err
	}

	// Optimization: call toString and ToLower once each
	strLower := strings.ToLower(toString(str))
	prefixLower := strings.ToLower(toString(prefix))
	return strings.HasPrefix(strLower, prefixLower), nil
}

// Optimized evalEndsWith - reduces string allocations
func evalEndsWithOptimized(args []Node, ctx *EvaluationContext) (bool, error) {
	if len(args) != 2 {
		return false, nil
	}

	str, err := evaluateNode(args[0], ctx)
	if err != nil {
		return false, err
	}
	suffix, err := evaluateNode(args[1], ctx)
	if err != nil {
		return false, err
	}

	// Optimization: call toString and ToLower once each
	strLower := strings.ToLower(toString(str))
	suffixLower := strings.ToLower(toString(suffix))
	return strings.HasSuffix(strLower, suffixLower), nil
}

// Optimized hashFiles using streaming hash (no full file read into memory)
func evalHashFilesOptimized(args []Node, ctx *EvaluationContext) (string, error) {
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

	hasher := sha256.New()
	hasContent := false

	for _, arg := range args {
		patternVal, err := evaluateNode(arg, ctx)
		if err != nil {
			return "", err
		}
		pattern := toString(patternVal)

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

			// Optimization: Use streaming read instead of ReadFile
			f, err := os.Open(match)
			if err != nil {
				continue
			}

			_, err = io.Copy(hasher, f)
			f.Close()
			if err != nil {
				continue
			}
			hasContent = true
		}
	}

	if !hasContent {
		return "", nil
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// Expression cache for parsed expressions
type ExpressionCache struct {
	mu    sync.RWMutex
	cache map[string]Node
}

func NewExpressionCache() *ExpressionCache {
	return &ExpressionCache{
		cache: make(map[string]Node),
	}
}

func (ec *ExpressionCache) Get(expr string) (Node, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	node, ok := ec.cache[expr]
	return node, ok
}

func (ec *ExpressionCache) Set(expr string, node Node) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.cache[expr] = node
}

// ParseExpressionCached parses with caching
func ParseExpressionCached(input string, cache *ExpressionCache) (Node, error) {
	if node, ok := cache.Get(input); ok {
		return node, nil
	}

	node, err := ParseExpression(input)
	if err != nil {
		return nil, err
	}

	cache.Set(input, node)
	return node, nil
}

// ============================================================
// BENCHMARKS COMPARING ORIGINAL VS OPTIMIZED
// ============================================================

// Direct comparison of evalContains implementations
func BenchmarkEvalContainsDirect(b *testing.B) {
	ctx := &EvaluationContext{
		Env: map[string]string{
			"VAR": "Hello World Example String for Testing Performance",
		},
	}

	args := []Node{
		&Identifier{Value: "env.VAR"},
		&StringLiteral{Value: "Example"},
	}

	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalContains(args, ctx)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalContainsOptimized(args, ctx)
		}
	})
}

// Direct comparison of evalStartsWith implementations
func BenchmarkEvalStartsWithDirect(b *testing.B) {
	ctx := &EvaluationContext{
		Env: map[string]string{
			"VAR": "refs/heads/feature/my-branch",
		},
	}

	args := []Node{
		&Identifier{Value: "env.VAR"},
		&StringLiteral{Value: "refs/heads/"},
	}

	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalStartsWith(args, ctx)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalStartsWithOptimized(args, ctx)
		}
	})
}

// Direct comparison of evalEndsWith implementations
func BenchmarkEvalEndsWithDirect(b *testing.B) {
	ctx := &EvaluationContext{
		Env: map[string]string{
			"VAR": "document.pdf",
		},
	}

	args := []Node{
		&Identifier{Value: "env.VAR"},
		&StringLiteral{Value: ".pdf"},
	}

	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalEndsWith(args, ctx)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalEndsWithOptimized(args, ctx)
		}
	})
}

// Benchmark hashFiles comparison
func BenchmarkHashFilesDirect(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "benchmark-hashfiles-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files (100KB total)
	for j := 0; j < 10; j++ {
		testFile := filepath.Join(tmpDir, "file"+string(rune('0'+j))+".txt")
		content := make([]byte, 10*1024)
		for i := range content {
			content[i] = byte((i + j) % 256)
		}
		if err := os.WriteFile(testFile, content, 0644); err != nil {
			b.Fatal(err)
		}
	}

	ctx := &EvaluationContext{
		WorkDir: tmpDir,
	}

	args := []Node{
		&StringLiteral{Value: "*.txt"},
	}

	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalHashFiles(args, ctx)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalHashFilesOptimized(args, ctx)
		}
	})
}

// Benchmark expression caching
func BenchmarkExpressionCaching(b *testing.B) {
	expr := "contains(env.GITHUB_REF, 'feature') && startsWith(env.GITHUB_ACTOR, 'dev')"

	b.Run("WithoutCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ParseExpression(expr)
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		cache := NewExpressionCache()
		for i := 0; i < b.N; i++ {
			ParseExpressionCached(expr, cache)
		}
	})
}

// Benchmark repeated evaluation with caching
func BenchmarkRepeatedEvaluationWithCache(b *testing.B) {
	env := map[string]string{
		"GITHUB_REF": "refs/heads/main",
	}
	condition := "success()"

	b.Run("WithoutCache", func(b *testing.B) {
		evaluator := NewConditionEvaluator()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 10; j++ {
				evaluator.Evaluate(condition, env, nil, true)
			}
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		cache := NewExpressionCache()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 10; j++ {
				// Parse with cache
				cond := strings.TrimSpace(condition)
				node, _ := ParseExpressionCached(cond, cache)

				// Evaluate (this part can't be cached as context changes)
				ctx := &EvaluationContext{
					Env:        env,
					JobSuccess: true,
				}
				result, _ := evaluateNode(node, ctx)
				_ = toBool(result)
			}
		}
	})
}

// Benchmark long string processing
func BenchmarkLongStringContains(b *testing.B) {
	// Create a 10KB string
	longStr := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 200)
	ctx := &EvaluationContext{
		Env: map[string]string{
			"LONG_VAR": longStr,
		},
	}

	args := []Node{
		&Identifier{Value: "env.LONG_VAR"},
		&StringLiteral{Value: "adipiscing"},
	}

	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalContains(args, ctx)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evalContainsOptimized(args, ctx)
		}
	})
}
