# 優先度中の課題 実装計画

**作成日:** 2025-12-26
**対象:** TEST_CASE_ANALYSIS.md の Medium Priority 項目

---

## 概要

以下の3つの課題に対する詳細な実装計画を記載します：

| # | 課題 | 対象ファイル | 影響範囲 |
|---|------|------------|---------|
| 1 | Command Execution Timeout | `internal/executor/` | 中 |
| 2 | Deeply Nested Expression Stack Overflow | `internal/expression/parser.go` | 低 |
| 3 | Context Cancellation Tests | `internal/util/`, `internal/worktree/` | 低 |

---

## 課題 1: Command Execution Timeout

### 現状分析

- `internal/executor/executor.go`: `Executor` インターフェースの `Execute` メソッドは `context.Context` を受け取っていない
- `internal/executor/host.go`: `exec.Command` を使用しており、タイムアウト制御ができない
- ユーザーが無限ループコマンドを実行した場合、手動でプロセスを kill する必要がある

### 実装計画

#### ステップ 1: Executor インターフェースの拡張

**ファイル:** `internal/executor/executor.go`

```go
// Executor defines the interface for command execution.
type Executor interface {
    // Execute runs the given command and returns the result.
    // The context can be used to cancel or timeout the command.
    Execute(ctx context.Context, config Config) (Result, error)
}
```

#### ステップ 2: Config に Timeout フィールドを追加

**ファイル:** `internal/executor/executor.go`

```go
// Config holds the configuration for command execution.
type Config struct {
    // Command is the shell command to execute.
    Command string
    // Env contains environment variables for the command.
    Env map[string]string
    // WorkingDir is the working directory for the command.
    WorkingDir string
    // Timeout is the maximum duration for command execution.
    // If zero, no timeout is applied (uses context deadline if set).
    Timeout time.Duration
}
```

#### ステップ 3: HostExecutor の実装更新

**ファイル:** `internal/executor/host.go`

```go
func (h *HostExecutor) Execute(ctx context.Context, config Config) (Result, error) {
    // Apply timeout if specified
    if config.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, config.Timeout)
        defer cancel()
    }

    cmd := exec.CommandContext(ctx, "sh", "-c", config.Command)
    // ... rest of implementation
}
```

#### ステップ 4: CLI フラグの追加

**ファイル:** `internal/cli/flags.go`

```go
type RunOptions struct {
    // ... existing fields
    Timeout time.Duration // Timeout for each step execution
}

// フラグ追加
fs.DurationVar(&opts.Timeout, "timeout", 30*time.Minute, "Timeout for each step")
fs.DurationVar(&opts.Timeout, "t", 30*time.Minute, "Timeout for each step (shorthand)")
```

#### ステップ 5: テストの追加

**ファイル:** `internal/executor/host_test.go`

```go
func TestHostExecutor_Execute_Timeout(t *testing.T) {
    executor := NewHostExecutor()
    ctx := context.Background()

    config := Config{
        Command: "sleep 10",
        Timeout: 100 * time.Millisecond,
    }

    start := time.Now()
    _, err := executor.Execute(ctx, config)
    elapsed := time.Since(start)

    // Should complete in approximately 100ms, not 10 seconds
    if elapsed > 500*time.Millisecond {
        t.Errorf("command should have timed out, took %v", elapsed)
    }

    // Error should indicate context deadline exceeded
    if err == nil || !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("expected DeadlineExceeded error, got %v", err)
    }
}

func TestHostExecutor_Execute_ContextCancellation(t *testing.T) {
    executor := NewHostExecutor()
    ctx, cancel := context.WithCancel(context.Background())

    // Cancel after short delay
    go func() {
        time.Sleep(100 * time.Millisecond)
        cancel()
    }()

    config := Config{
        Command: "sleep 10",
    }

    _, err := executor.Execute(ctx, config)
    if err == nil || !errors.Is(err, context.Canceled) {
        t.Errorf("expected Canceled error, got %v", err)
    }
}
```

### 影響を受けるファイル

1. `internal/executor/executor.go` - インターフェース変更
2. `internal/executor/host.go` - 実装変更
3. `internal/executor/host_test.go` - テスト追加
4. `internal/cli/flags.go` - フラグ追加
5. `internal/cli/run.go` - Execute呼び出し更新
6. `internal/cli/step_executor.go` - Execute呼び出し更新

### 破壊的変更

- `Executor.Execute()` のシグネチャ変更は破壊的変更
- 既存の呼び出し箇所を全て更新する必要あり

---

## 課題 2: Deeply Nested Expression Stack Overflow

### 現状分析

- `internal/expression/parser.go`: 再帰的なパース処理に深度制限がない
- `parseExpression()` → `parseOrExpr()` → `parseAndExpr()` → `parseUnaryExpr()` → `parseComparisonExpr()` → `parsePrimaryExpr()` と再帰
- 悪意のある/誤った式 `(((((...))))` でスタックオーバーフローの可能性

### 実装計画

#### ステップ 1: Parser に深度追跡を追加

**ファイル:** `internal/expression/parser.go`

```go
const (
    // MaxParseDepth is the maximum nesting depth for expressions.
    MaxParseDepth = 50
)

// Parser parses tokens into an Abstract Syntax Tree.
type Parser struct {
    tokenizer *Tokenizer
    curToken  Token
    peekToken Token
    errors    []string
    depth     int  // 追加: 現在のネスト深度
}

// ErrMaxDepthExceeded is returned when expression nesting exceeds the limit.
var ErrMaxDepthExceeded = errors.New("maximum expression depth exceeded")
```

#### ステップ 2: 深度チェックの実装

**ファイル:** `internal/expression/parser.go`

```go
// enterDepth increments the parse depth and returns an error if limit exceeded.
func (p *Parser) enterDepth() error {
    p.depth++
    if p.depth > MaxParseDepth {
        return ErrMaxDepthExceeded
    }
    return nil
}

// exitDepth decrements the parse depth.
func (p *Parser) exitDepth() {
    p.depth--
}
```

#### ステップ 3: パース関数の更新

主要なエントリポイントに深度チェックを追加：

```go
func (p *Parser) parsePrimaryExpr() Node {
    if err := p.enterDepth(); err != nil {
        p.errors = append(p.errors, err.Error())
        return nil
    }
    defer p.exitDepth()

    switch p.curToken.Type {
    case TOKEN_LPAREN:
        p.nextToken() // consume '('
        expr := p.parseExpression()  // 再帰呼び出し
        // ...
    }
    // ...
}
```

#### ステップ 4: テストの追加

**ファイル:** `internal/expression/parser_test.go`

```go
func TestParser_Parse_MaxDepth(t *testing.T) {
    // Create deeply nested expression: (((((...true...)))))
    depth := 100
    input := strings.Repeat("(", depth) + "true" + strings.Repeat(")", depth)

    _, err := ParseExpression(input)
    if err == nil {
        t.Error("expected error for deeply nested expression")
    }

    if !strings.Contains(err.Error(), "maximum expression depth exceeded") {
        t.Errorf("expected depth error, got: %v", err)
    }
}

func TestParser_Parse_AtMaxDepth(t *testing.T) {
    // Create expression at exactly max depth (should succeed)
    depth := MaxParseDepth
    input := strings.Repeat("(", depth) + "true" + strings.Repeat(")", depth)

    _, err := ParseExpression(input)
    if err != nil {
        t.Errorf("expression at max depth should succeed, got: %v", err)
    }
}

func TestParser_Parse_DeeplyNestedNot(t *testing.T) {
    // Test deeply nested NOT expressions: !!!!!!!...true
    depth := 100
    input := strings.Repeat("!", depth) + "true"

    _, err := ParseExpression(input)
    if err == nil {
        t.Error("expected error for deeply nested NOT expression")
    }
}
```

### 影響を受けるファイル

1. `internal/expression/parser.go` - 深度追跡の実装
2. `internal/expression/parser_test.go` - テスト追加

### 破壊的変更

- なし（パース失敗は既存のエラー処理で対応）

---

## 課題 3: Context Cancellation Tests

### 現状分析

- `internal/util/git.go`: `exec.CommandContext` を使用しており、context サポートは既に実装済み
- `internal/worktree/worktree.go`: 同様に `exec.CommandContext` を使用
- テストでキャンセル/タイムアウト動作を検証していない

### 実装計画

#### ステップ 1: git.go のコンテキストキャンセルテスト

**ファイル:** `internal/util/git_test.go`

```go
func TestFindGitRoot_ContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // 即座にキャンセル

    _, err := FindGitRoot(ctx, "/some/path")
    if err == nil {
        t.Error("expected error for cancelled context")
    }
}

func TestFindGitRoot_ContextTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
    defer cancel()

    // Wait for timeout
    time.Sleep(10 * time.Millisecond)

    _, err := FindGitRoot(ctx, "/some/path")
    if err == nil {
        t.Error("expected error for timed out context")
    }
}

func TestGitHeadSHA_ContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel()

    repoRoot := findTestRepoRoot(t, ".")
    _, err := GitHeadSHA(ctx, repoRoot)
    if err == nil {
        t.Error("expected error for cancelled context")
    }
}
```

#### ステップ 2: worktree.go のコンテキストキャンセルテスト

**ファイル:** `internal/worktree/worktree_test.go`

```go
func TestCreateWorkspace_ContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // 即座にキャンセル

    repoRoot := findTestRepoRoot(t)
    _, err := CreateWorkspace(ctx, repoRoot, false)
    if err == nil {
        t.Error("expected error for cancelled context")
    }
}

func TestCreateWorkspace_ContextTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
    defer cancel()

    time.Sleep(10 * time.Millisecond)

    repoRoot := findTestRepoRoot(t)
    _, err := CreateWorkspace(ctx, repoRoot, false)
    if err == nil {
        t.Error("expected error for timed out context")
    }
}

func TestRemoveWorkspace_ContextCancellation(t *testing.T) {
    // First create a workspace with valid context
    ctx := context.Background()
    repoRoot := findTestRepoRoot(t)
    ws, err := CreateWorkspace(ctx, repoRoot, false)
    if err != nil {
        t.Fatalf("failed to create workspace: %v", err)
    }

    // Then try to remove with cancelled context
    cancelledCtx, cancel := context.WithCancel(context.Background())
    cancel()

    err = RemoveWorkspace(cancelledCtx, ws)
    // Note: RemoveWorkspace may succeed via manual cleanup fallback
    // Just verify the function handles the cancelled context gracefully
    t.Logf("RemoveWorkspace with cancelled context returned: %v", err)

    // Ensure cleanup
    _ = os.RemoveAll(ws.Path)
}
```

### 影響を受けるファイル

1. `internal/util/git_test.go` - テスト追加
2. `internal/worktree/worktree_test.go` - テスト追加

### 破壊的変更

- なし（テストのみの追加）

---

## 実装優先順位

| 順位 | 課題 | 理由 |
|-----|------|------|
| 1 | Context Cancellation Tests | 破壊的変更なし、既存実装の検証のみ |
| 2 | Deeply Nested Expression | 破壊的変更なし、セキュリティ改善 |
| 3 | Command Execution Timeout | 破壊的変更あり、慎重な実装が必要 |

---

## 推定作業量

| 課題 | ファイル数 | 作業内容 |
|-----|----------|---------|
| Context Cancellation Tests | 2 | テスト追加のみ |
| Deeply Nested Expression | 2 | 実装変更 + テスト追加 |
| Command Execution Timeout | 6+ | インターフェース変更 + 実装 + テスト |

---

## チェックリスト

### 課題 1: Command Execution Timeout
- [ ] `executor.go` - Executor インターフェースに context 追加
- [ ] `executor.go` - Config に Timeout フィールド追加
- [ ] `host.go` - Execute メソッドの context 対応
- [ ] `host_test.go` - タイムアウトテスト追加
- [ ] `host_test.go` - コンテキストキャンセルテスト追加
- [ ] `flags.go` - --timeout フラグ追加
- [ ] 全ての Execute 呼び出し箇所を更新
- [ ] ドキュメント更新

### 課題 2: Deeply Nested Expression
- [ ] `parser.go` - MaxParseDepth 定数追加
- [ ] `parser.go` - Parser に depth フィールド追加
- [ ] `parser.go` - enterDepth/exitDepth メソッド追加
- [ ] `parser.go` - parsePrimaryExpr に深度チェック追加
- [ ] `parser.go` - parseUnaryExpr に深度チェック追加
- [ ] `parser_test.go` - 深度制限テスト追加
- [ ] `parser_test.go` - 境界値テスト追加

### 課題 3: Context Cancellation Tests
- [ ] `git_test.go` - FindGitRoot キャンセルテスト追加
- [ ] `git_test.go` - GitHeadSHA キャンセルテスト追加
- [ ] `git_test.go` - GitHeadRef キャンセルテスト追加
- [ ] `worktree_test.go` - CreateWorkspace キャンセルテスト追加
- [ ] `worktree_test.go` - RemoveWorkspace キャンセルテスト追加
