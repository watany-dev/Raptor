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
- **シンプル**: `run:`ステップに特化した設計

```bash
# インストール（Go環境がある場合）
go install github.com/watany-dev/raptor/cmd/raptor@latest

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
=== Workflow: CI ===

--- Job: test (Test) ---
  Step 1: [checkout] Checkout code
  Step 2: [setup-go] Set up Go
  Step 3: [test] Run tests
    Command:
      go test -v ./...

--- Job: lint (Lint) ---
  Step 1: [checkout] Checkout code
  Step 2: [lint] Run golangci-lint
    Command:
      golangci-lint run
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

        # 条件分岐
        if: ${{ success() }}

        # 実行コマンド
        run: |
          echo "Hello from Raptor!"
          echo "GLOBAL_VAR=$GLOBAL_VAR"

        # 作業ディレクトリ
        working-directory: ./src
```

**サポートされる`if`条件:**
- `true` / `false`（リテラル）
- `success()` / `failure()` / `always()`
- `${{ env.VAR == 'value' }}` / `${{ env.VAR != 'value' }}`
- `${{ steps.step_id.outcome == 'success' }}`

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
│   ├── expression/          # if条件評価
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
            ├── runJob: ジョブ実行
            │   ├── [step_executor.go] 各ステップ実行
            │   │   ├── if条件評価
            │   │   ├── コマンド実行
            │   │   └── 環境ファイル解析
            │   └── 環境変数マージ
            └── RemoveWorkspace: クリーンアップ
```

### 1. ワークフロー解析（YAML処理）

`internal/workflow/load.go`:

```go
type WorkflowFile struct {
    Name     string            `yaml:"name"`
    Env      map[string]string `yaml:"env"`
    Jobs     map[string]Job    `yaml:"jobs"`
    JobOrder []string          `yaml:"-"` // 定義順序を保持
}

type Job struct {
    Name   string
    RunsOn string            `yaml:"runs-on"`
    Env    map[string]string
    Steps  []Step
}

type Step struct {
    ID               string
    Name             string
    If               string
    Run              string
    Env              map[string]string
    WorkingDirectory string `yaml:"working-directory"`
}
```

**最適化ポイント:**
- YAMLを一度だけパースし、`yaml.Node`から構造体とジョブ順序を同時に抽出
- マップのジョブ順序を維持するため、`JobOrder`スライスを使用

### 2. Git Worktreeによる隔離実行

Raptorの最大の特徴は、**Git worktreeを使った隔離実行**です。

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

### 3. 環境変数の階層化マージ

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

### 4. 条件分岐（if式）評価

`internal/expression/evaluator.go`:

```go
var (
    // 正規表現をプリコンパイル（パフォーマンス最適化）
    envComparePattern = regexp.MustCompile(`\$\{\{\s*env\.(\w+)\s*(==|!=)\s*'([^']*)'\s*\}\}`)
    stepOutcomePattern = regexp.MustCompile(`\$\{\{\s*steps\.(\w+)\.outcome\s*(==|!=)\s*'([^']*)'\s*\}\}`)
)

func EvaluateCondition(condition string, ctx *StepContext) bool {
    condition = strings.TrimSpace(condition)

    switch condition {
    case "", "true":
        return true
    case "false":
        return false
    case "always()":
        return true
    case "success()":
        return ctx.PreviousStepSuccess
    case "failure()":
        return !ctx.PreviousStepSuccess
    }

    // 環境変数比較: ${{ env.VAR == 'value' }}
    if matches := envComparePattern.FindStringSubmatch(condition); matches != nil {
        varName, operator, expected := matches[1], matches[2], matches[3]
        actual := ctx.Env[varName]
        if operator == "==" {
            return actual == expected
        }
        return actual != expected
    }

    // ステップ結果参照: ${{ steps.step_id.outcome == 'success' }}
    // ...

    return true // デフォルトは実行
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
    // 名前・値のバリデーション
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

### Raptorを選ぶべき理由

1. **シンプルさ**: Docker不要、単一バイナリ
2. **高速性**: コンテナ起動のオーバーヘッドなし
3. **セキュリティ**: Git worktree隔離、環境変数保護
4. **使いやすさ**: 直感的なCLI、ドライランモード

### 今後の展望

- テストカバレッジ100%達成（現在75.1%）
- ドキュメントの充実
- より詳細なエラーメッセージ

GitHub Actionsのワークフローをローカルでテストしたい方は、ぜひRaptorを試してみてください！

---

## リンク

- [GitHub リポジトリ](https://github.com/watany-dev/raptor)
- [リリースページ](https://github.com/watany-dev/raptor/releases)

---

*この記事がお役に立ちましたら、GitHubでスターをお願いします！*
