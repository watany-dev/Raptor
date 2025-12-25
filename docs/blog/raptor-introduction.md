# Raptor: Docker不要でGitHub Actionsをローカル実行する軽量CLIツール

GitHub Actionsのワークフローをローカルで実行してデバッグしたいと思ったことはありませんか？`git push`してCIが失敗し、修正してまたpush...という繰り返しは開発効率を大きく下げます。

本記事では、GitHub Actionsをローカル環境で実行できる軽量CLIツール「**Raptor**」を紹介します。nektos/actとの違いや、内部実装の詳細まで解説していきます。

## 目次

1. [Raptorとは](#raptorとは)
2. [基本的な使い方](#基本的な使い方)
3. [実装の詳細解説](#実装の詳細解説)
4. [nektos/actとの比較](#nektosactとの比較)
5. [まとめ](#まとめ)

---

## Raptorとは

Raptorは、GitHub Actionsのワークフローファイル（`.github/workflows/*.yml`）をローカル環境で実行するための**軽量CLIツール**です。

### 主な特徴

- **Docker不要**: ホストシステムで直接実行
- **軽量**: 依存関係は`gopkg.in/yaml.v3`のみ
- **セキュア**: Git worktreeによる隔離実行
- **高速**: コンテナ起動のオーバーヘッドなし
- **高機能な条件式**: AND/OR/NOT、文字列関数、hashFilesをフルサポート

```bash
# インストール（Go環境がある場合）
go install github.com/watany-dev/raptor/cmd/raptor@v0.2.0

# または、リリースからバイナリをダウンロード
```

---

## 基本的な使い方

### ワークフローの確認（ドライラン）

まずは実行せずに、ワークフローの内容を確認しましょう。

```bash
# ワークフローファイルを指定してドライラン
raptor -w .github/workflows/ci.yml
```

出力例:
```
🔍 DRY RUN MODE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Workflow: .github/workflows/ci.yml
Name: CI
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 Job: build
   Runs-on: ubuntu-latest

   [1] Setup
       Command:
         echo "Setting up..."

   [2] Build
       Command:
         echo "Building..."

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
To execute this workflow, use: raptor run -w .github/workflows/ci.yml
```

### ワークフローの実行

```bash
# ワークフロー全体を実行
raptor run -w .github/workflows/ci.yml

# 特定のジョブのみ実行
raptor run -w .github/workflows/ci.yml -j test

# 実行前に確認（--dry-run）
raptor run -w .github/workflows/ci.yml --dry-run
```

### コマンドオプション

| オプション | 短縮形 | 説明 |
|--------|-------|-------------|
| `--workflow` | `-w` | ワークフローファイルへのパス（必須） |
| `--job` | `-j` | 実行するジョブID（省略時は全ジョブ実行） |
| `--workdir` | `-C` | 作業ディレクトリ（デフォルト: カレントディレクトリ） |
| `--dry-run` | `-n` | 実行せずにプレビュー |
| `--ignore-if-errors` | | 条件評価エラーを無視（レガシーモード） |

### サポートされる機能

Raptorは以下のワークフロー機能をサポートしています：

```yaml
name: Example Workflow

# ワークフロー全体の環境変数
env:
  GLOBAL_VAR: "global-value"

jobs:
  example:
    runs-on: ubuntu-latest

    # ジョブレベルの環境変数
    env:
      JOB_VAR: "job-value"

    steps:
      - name: Step with conditions
        # ステップレベルの環境変数
        env:
          STEP_VAR: "step-value"

        # 条件分岐（AND/OR/NOT、文字列関数も対応）
        if: ${{ success() && env.DEPLOY_ENV == 'production' }}

        # 実行コマンド
        run: |
          echo "Hello from Raptor!"
          echo "GLOBAL_VAR=$GLOBAL_VAR"

        # 作業ディレクトリ
        working-directory: ./src
```

**サポートされる`if`条件:**

| 構文 | 説明 |
|--------|-------------|
| `true` / `false` | リテラルブール値 |
| `success()` | 前のステップがすべて成功 |
| `failure()` | いずれかのステップが失敗 |
| `always()` | 常に実行（失敗後も続行） |
| `cancelled()` | キャンセル時に実行（常にfalse） |
| `${{ env.VAR == 'value' }}` | 環境変数の比較 |
| `${{ env.VAR != 'value' }}` | 環境変数の否定 |
| `${{ steps.ID.outcome == 'success' }}` | ステップ結果の参照 |
| `${{ expr1 && expr2 }}` | 論理AND演算子 |
| `${{ expr1 \|\| expr2 }}` | 論理OR演算子 |
| `${{ !expr }}` | 論理NOT演算子 |
| `${{ (expr) }}` | 括弧によるグループ化 |
| `contains(search, item)` | 文字列/配列に値が含まれるか |
| `startsWith(search, prefix)` | 文字列がプレフィックスで始まるか |
| `endsWith(search, suffix)` | 文字列がサフィックスで終わるか |
| `hashFiles(pattern, ...)` | パターンにマッチするファイルのSHA-256ハッシュ |

### GITHUB_ENV / GITHUB_OUTPUT の使用

```yaml
steps:
  - name: Set environment variable
    run: |
      echo "MY_VAR=hello" >> $GITHUB_ENV

  - name: Use the variable
    run: |
      echo "MY_VAR is: $MY_VAR"
```

Raptorは`GITHUB_ENV`、`GITHUB_PATH`、`GITHUB_OUTPUT`のマルチライン形式もサポートしています。

---

## 実装の詳細解説

Raptorの内部実装を詳しく見ていきましょう。

### プロジェクト構成

```
raptor/
├── cmd/raptor/
│   └── main.go              # CLIエントリポイント
├── internal/
│   ├── cli/                 # CLI処理
│   │   ├── run.go           # ワークフロー実行ロジック
│   │   ├── flags.go         # フラグ解析
│   │   ├── dry_run.go       # ドライラン表示
│   │   └── step_executor.go # ステップ実行エンジン
│   ├── envfiles/            # GITHUB_ENV等の解析
│   ├── executor/            # コマンド実行エンジン
│   ├── expression/          # 条件式評価（ASTベース）
│   │   ├── tokenizer.go     # 字句解析
│   │   ├── parser.go        # 構文解析
│   │   ├── ast.go           # 抽象構文木
│   │   └── evaluator.go     # 評価エンジン
│   ├── runtime/             # 環境変数処理
│   ├── security/            # セキュリティ検証
│   ├── workflow/            # YAML解析
│   └── worktree/            # Git worktree管理
└── go.mod
```

### 実行フロー

```
ユーザー入力
    ↓
[flags.go] コマンドライン解析
    ↓
[run.go] 実行開始
    ├── FindGitRoot: リポジトリルート検出
    ├── CreateWorkspace: Git worktree生成（隔離環境）
    ├── LoadWorkflowFile: YAML解析
    └── executeJobs: ジョブ実行
        └── for each job:
            ├── [step_executor.go] ステップ実行
            │   ├── if条件評価（ASTベース）
            │   ├── コマンド実行
            │   └── 環境ファイル解析
            └── RemoveWorkspace: クリーンアップ
```

### 1. 条件式評価エンジン（ASTベース実装）

Raptorの最大の技術的特徴は、**完全なAST（抽象構文木）ベースの条件式評価エンジン**です。

#### なぜASTベースなのか？

初期実装では正規表現ベースで条件式を評価していましたが、以下の問題がありました：

1. 複雑な条件式（ネストした括弧、複数の演算子）の処理が困難
2. 正規表現の組み合わせ爆発による保守性の低下
3. エラーメッセージの品質が低い

ASTベースの実装により、これらの問題を解決しました。

#### トークナイザー（字句解析）

`internal/expression/tokenizer.go`:

```go
type TokenType int

const (
    TOKEN_EOF TokenType = iota
    TOKEN_LPAREN   // (
    TOKEN_RPAREN   // )
    TOKEN_AND      // &&
    TOKEN_OR       // ||
    TOKEN_NOT      // !
    TOKEN_EQ       // ==
    TOKEN_NE       // !=
    TOKEN_COMMA    // ,
    TOKEN_STRING   // 'value'
    TOKEN_IDENT    // env.VAR, steps.id.outcome
    TOKEN_TRUE     // true
    TOKEN_FALSE    // false
)
```

トークナイザーは入力文字列をトークンの列に変換します：

```
入力: success() && env.DEPLOY == 'prod'
  ↓
トークン列: [IDENT:success, LPAREN, RPAREN, AND, IDENT:env.DEPLOY, EQ, STRING:prod]
```

#### 抽象構文木（AST）

`internal/expression/ast.go`:

```go
// Node represents a node in the Abstract Syntax Tree.
type Node interface {
    node()
}

// BinaryExpr represents a binary expression (&&, ||, ==, !=).
type BinaryExpr struct {
    Left     Node
    Operator TokenType
    Right    Node
}

// UnaryExpr represents a unary expression (!).
type UnaryExpr struct {
    Operator TokenType
    Operand  Node
}

// CallExpr represents a function call (e.g., success(), contains(a, b)).
type CallExpr struct {
    FuncName  string
    Arguments []Node
}

// Identifier represents an identifier (e.g., env.VAR, steps.id.outcome).
type Identifier struct {
    Value string
}

// StringLiteral represents a string literal (e.g., 'value').
type StringLiteral struct {
    Value string
}

// BoolLiteral represents a boolean literal (true, false).
type BoolLiteral struct {
    Value bool
}
```

#### パーサー（構文解析）

パーサーはトークン列をASTに変換します。演算子の優先順位を正しく処理するため、**再帰下降パーサー**を採用しています：

```
入力: success() && !failure() || env.DEBUG == 'true'
  ↓
AST:
        BinaryExpr(||)
       /            \
  BinaryExpr(&&)   BinaryExpr(==)
   /        \        /        \
CallExpr  UnaryExpr  Identifier  StringLiteral
(success)    |      (env.DEBUG)   ('true')
          CallExpr
         (failure)
```

#### 評価エンジン

`internal/expression/evaluator.go`:

```go
// ConditionEvaluator evaluates step if conditions.
type ConditionEvaluator struct {
    // StrictMode controls error handling behavior.
    StrictMode bool

    // cache stores parsed expressions to avoid re-parsing.
    // This provides ~97% time reduction for repeated evaluations.
    cache   map[string]Node
    cacheMu sync.RWMutex
}
```

**パフォーマンス最適化:**

同じ条件式が複数回評価される場合（例：複数のステップで`success()`を使用）、パース結果をキャッシュすることで約97%の時間削減を実現しています。

```go
func (ce *ConditionEvaluator) parseWithCache(expr string) (Node, error) {
    // Try read from cache first
    ce.cacheMu.RLock()
    if node, ok := ce.cache[expr]; ok {
        ce.cacheMu.RUnlock()
        return node, nil
    }
    ce.cacheMu.RUnlock()

    // Parse the expression
    node, err := ParseExpression(expr)
    if err != nil {
        return nil, err
    }

    // Store in cache
    ce.cacheMu.Lock()
    ce.cache[expr] = node
    ce.cacheMu.Unlock()

    return node, nil
}
```

#### 短絡評価（Short-circuit Evaluation）

論理演算子`&&`と`||`は短絡評価をサポートしています：

```go
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
```

これにより、不要な評価を避けてパフォーマンスを向上させています。

#### hashFiles関数の実装

`hashFiles()`はファイルのSHA-256ハッシュを計算します：

```go
func evalHashFiles(args []Node, ctx *EvaluationContext) (string, error) {
    var allBytes []byte
    for _, arg := range args {
        pattern := toString(patternVal)

        // Glob patterns for file matching
        matches, _ := filepath.Glob(filepath.Join(workDir, pattern))

        // Sort for consistent hashing
        sort.Strings(matches)

        for _, match := range matches {
            data, _ := os.ReadFile(match)
            allBytes = append(allBytes, data...)
        }
    }

    hash := sha256.Sum256(allBytes)
    return hex.EncodeToString(hash[:]), nil
}
```

### 2. Git Worktreeによる隔離実行

Raptorの重要な特徴は、**Git worktreeを使った隔離実行**です。

`internal/worktree/worktree.go`:

```go
func CreateWorkspace(ctx context.Context, repoRoot string, verified bool) (*Workspace, error) {
    // 一意なIDを生成
    id := generateID()

    // .raptor/ws-<id> ディレクトリを作成
    wsPath := filepath.Join(repoRoot, ".raptor", "ws-"+id)

    // git worktree add --detach wsPath
    cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", wsPath)
    cmd.Dir = repoRoot

    return &Workspace{Path: wsPath, ID: id}, nil
}

func RemoveWorkspace(ctx context.Context, ws *Workspace) error {
    // git worktree remove --force wsPath
    return exec.CommandContext(ctx, "git", "worktree", "remove", "--force", ws.Path).Run()
}
```

**なぜGit worktreeなのか？**

1. **メインリポジトリの保護**: ワークフロー実行がメインの作業ディレクトリに影響しない
2. **クリーンな状態**: 毎回クリーンな状態から実行
3. **簡単なクリーンアップ**: 失敗しても`git worktree remove`で完全削除
4. **Dockerより軽量**: コンテナ起動のオーバーヘッドなし

### 3. ステップ実行エンジン

`internal/cli/step_executor.go`:

```go
// StepExecutor handles the execution of individual workflow steps.
type StepExecutor struct {
    executor     executor.Executor
    evaluator    *expression.ConditionEvaluator
    stdout       io.Writer
    stderr       io.Writer
    workDir      string
    envFilePath  string
    pathFilePath string
}

// Execute executes a single step and returns the result.
func (se *StepExecutor) Execute(step *workflow.Step, index int, ctx *ExecutionContext) (*StepResult, error) {
    // Merge step-level env
    stepEnv := runtime.MergeEnv(ctx.AccumulatedEnv, step.Env)

    // Evaluate if condition with AST-based evaluator
    shouldRun, err := se.evaluator.EvaluateWithWorkDir(
        step.If, stepEnv, ctx.StepsContext, ctx.JobSuccess, se.workDir)

    if err != nil {
        if !shouldRun {
            // Strict mode - fail the step
            return nil, fmt.Errorf("condition evaluation failed: %w", err)
        }
        // Permissive mode - log warning and continue
        fmt.Fprintf(se.stderr, "Warning: %v\n", err)
    }

    if !shouldRun {
        return se.handleSkippedStep(step, index, stepName, ctx)
    }

    return se.executeStep(step, index, stepName, stepEnv, ctx)
}
```

**StrictModeの動作:**

- `StrictMode=true`（デフォルト）: 条件式のエラーでワークフローを停止
- `StrictMode=false`（`--ignore-if-errors`）: 警告を出力して続行

### 4. 環境変数の階層化マージ

GitHub Actionsでは、環境変数が複数レベルで定義できます。Raptorはこれを正しく処理します。

`internal/runtime/defaults.go`:

```go
// 優先度（後の方が上書き）
// 1. Raptorのデフォルト < 2. ワークフロー < 3. ジョブ < 4. ステップ
mergedEnv := MergeEnv(
    runtime.DefaultBaseEnv(),    // CI=true, GITHUB_ACTIONS=true など
    wf.Env,                      // ワークフロー全体
    job.Env,                     // ジョブレベル
    step.Env,                    // ステップレベル（最高優先度）
)
```

デフォルト環境変数:
```go
map[string]string{
    "CI":               "true",
    "GITHUB_ACTIONS":   "true",
    "GITHUB_WORKSPACE": workspacePath,
    "GITHUB_SHA":       gitHeadSHA,
    "GITHUB_REF":       gitHeadRef,
}
```

### 5. セキュリティ機構

Raptorはセキュリティを重視した設計になっています。

#### 環境変数ブロックリスト

`internal/security/envvar.go`:

```go
var BlockedEnvVars = map[string]string{
    "LD_PRELOAD":            "Library injection attack",
    "LD_LIBRARY_PATH":       "Library path hijacking",
    "DYLD_INSERT_LIBRARIES": "macOS library injection",
    "BASH_ENV":              "Shell startup script injection",
    "ENV":                   "Shell startup script injection",
    "IFS":                   "Command parsing manipulation",
    "GIT_DIR":               "Git directory hijacking",
    "GIT_WORK_TREE":         "Git worktree hijacking",
    // ...
}

func ValidateEnvVar(name, value string) error {
    if reason, blocked := BlockedEnvVars[name]; blocked {
        return fmt.Errorf("blocked environment variable %s: %s", name, reason)
    }
    return nil
}
```

#### パストラバーサル防止

`internal/security/path.go`:

```go
func ValidateWorkingDirectory(workDir, basePath string) error {
    // 絶対パス禁止
    if filepath.IsAbs(workDir) {
        return errors.New("absolute path not allowed")
    }

    // パストラバーサル防止
    cleaned := filepath.Clean(workDir)
    if strings.HasPrefix(cleaned, "..") {
        return errors.New("path traversal detected")
    }

    // ワークスペース外への逃脱防止
    fullPath := filepath.Join(basePath, workDir)
    relPath, _ := filepath.Rel(basePath, fullPath)
    if strings.HasPrefix(relPath, "..") {
        return errors.New("path escapes workspace")
    }

    return nil
}
```

### 6. コマンド実行エンジン

`internal/executor/host.go`:

```go
type HostExecutor struct {
    cachedSysEnv []string   // システム環境のキャッシュ
    once         sync.Once
}

func (h *HostExecutor) Execute(config Config) (Result, error) {
    // システム環境を一度だけ取得（パフォーマンス最適化）
    h.once.Do(func() {
        h.cachedSysEnv = os.Environ()
    })

    // sh -c でコマンド実行
    cmd := exec.Command("sh", "-c", config.Command)
    cmd.Dir = config.WorkingDir
    cmd.Env = mergeEnvironment(h.cachedSysEnv, config.Env)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()

    return Result{
        ExitCode: cmd.ProcessState.ExitCode(),
        Stdout:   stdout.String(),
        Stderr:   stderr.String(),
    }, err
}
```

**インターフェース設計:**

```go
type Executor interface {
    Execute(config Config) (Result, error)
}
```

この設計により、将来的に`DockerExecutor`などの実装を追加することが容易です。

---

## nektos/actとの比較

### 概要比較

| 特性 | nektos/act | Raptor |
|------|-----------|--------|
| **実行環境** | Dockerコンテナ | ホストシステム直接 |
| **サポート範囲** | ほぼ完全 | `run:`ステップのみ |
| **依存関係** | Docker必須 | なし（スタンドアロン） |
| **起動速度** | 遅い（コンテナ起動） | 高速（プロセス直接） |
| **バイナリサイズ** | N/A（Docker + イメージ） | 約10-15MB |
| **隔離方式** | コンテナ隔離 | Git worktree隔離 |
| **条件式サポート** | 完全 | AND/OR/NOT、文字列関数、hashFiles対応 |

### Raptorのメリット

#### 1. Docker不要
```bash
# act の場合
docker pull catthehacker/ubuntu:act-latest
act

# Raptor の場合
raptor run -w .github/workflows/ci.yml  # すぐ実行可能
```

CI環境にDockerがない、またはDockerのセットアップが面倒な場合に最適です。

#### 2. 高速な実行
```bash
# act: コンテナ起動に数秒〜十数秒
# Raptor: 即座に実行開始
```

コンテナのプル・起動時間がないため、素早いフィードバックが得られます。

#### 3. ホスト環境の利用
```yaml
steps:
  - run: |
      # ホストのツールがそのまま使える
      node --version
      go version
      rustc --version
```

ローカル環境のツールをそのまま使えるため、環境の差異による問題が少ないです。

#### 4. 軽量
```bash
# Raptor: 単一バイナリ、約10-15MB
# act: Docker + イメージ（数GB〜）
```

### Raptorのデメリット

#### 1. 機能制限

```yaml
# ❌ サポートされない: uses（外部アクション）
steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5

# ❌ サポートされない: matrix（マトリックスビルド）
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest]

# ❌ サポートされない: needs（ジョブ依存関係）
jobs:
  build:
    needs: test
```

#### 2. 環境の違い

本番のGitHub Actions環境（Ubuntu on Azure）とローカル環境で差異が出る可能性があります。

### 使い分けガイド

**Raptor向きのケース:**
- `run:`ステップのみのシンプルなワークフロー
- ローカルテスト・デバッグを高速に行いたい
- Docker環境がない/使いたくない
- 軽量なツールを好む

**nektos/act向きのケース:**
- `uses:`で外部アクションを使用
- 完全なGitHub Actions互換性が必要
- Dockerが利用可能
- matrixビルドが必要

---

## まとめ

Raptorは、GitHub Actionsの`run:`ステップをローカルで実行するためのシンプルで軽量なツールです。

### 技術的ハイライト

1. **ASTベースの式評価エンジン**: 完全な構文解析による高精度な条件評価
2. **パースキャッシュ**: 同一条件式の再評価で約97%の時間削減
3. **短絡評価**: AND/OR演算子の効率的な評価
4. **Git worktree隔離**: Docker不要でセキュアな隔離実行

### Raptorを選ぶべき理由

1. **シンプルさ**: Docker不要、単一バイナリ
2. **高速性**: コンテナ起動のオーバーヘッドなし
3. **セキュリティ**: Git worktree隔離、環境変数保護
4. **高機能な条件式**: AND/OR/NOT、文字列関数、hashFilesをフルサポート

GitHub Actionsのワークフローをローカルでテストしたい方は、ぜひRaptorを試してみてください！

---

## リンク

- [GitHub リポジトリ](https://github.com/watany-dev/raptor)
- [リリースページ](https://github.com/watany-dev/raptor/releases)

---

*この記事がお役に立ちましたら、GitHubでスターをお願いします！*
