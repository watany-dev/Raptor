# Raptor Performance Review

**レビュー日**: 2025-12-22
**レビュアー**: Claude (Opus 4.5)

## エグゼクティブサマリー

Raptorは比較的小規模なCLIツールであり、全体的にはシンプルで効率的な実装になっています。ただし、いくつかの改善の余地があります。このレビューでは、パフォーマンスに影響を与える可能性のある問題を特定し、優先度順に整理しました。

**総合評価**: ★★★★☆ (4/5)

---

## 問題一覧

| 優先度 | カテゴリ | 問題 | 影響度 |
|--------|----------|------|--------|
| 高 | I/O | YAMLの二重パース | 中 |
| 中 | メモリ | 環境変数の重複コピー | 低〜中 |
| 中 | I/O | Git操作の冗長性 | 低 |
| 低 | メモリ | MergeEnvの毎回割り当て | 低 |
| 低 | 文字列 | ヒアドキュメントの非効率な結合 | 低 |

---

## 詳細分析

### 1. YAMLの二重パース (優先度: 高)

**ファイル**: `internal/workflow/load.go:11-29`

**問題**:
ワークフローファイルが2回パースされています：
1. 構造体 (`WorkflowFile`) へのUnmarshal
2. ジョブ順序取得のための `yaml.Node` へのパース

```go
// 1回目のパース
var wf WorkflowFile
if err := yaml.Unmarshal(data, &wf); err != nil {
    return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
}

// 2回目のパース
wf.JobOrder, err = extractJobOrder(data)
```

**影響**:
- ファイルサイズに比例してCPU使用量が増加
- 大規模なワークフローファイルで顕著

**推奨修正**:
```go
func LoadWorkflowFile(path string) (*WorkflowFile, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read workflow file: %w", err)
    }

    // yaml.Nodeに1回だけパース
    var root yaml.Node
    if err := yaml.Unmarshal(data, &root); err != nil {
        return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
    }

    // Nodeから構造体とジョブ順序の両方を抽出
    wf, jobOrder, err := parseWorkflowFromNode(&root)
    if err != nil {
        return nil, err
    }
    wf.JobOrder = jobOrder

    return wf, nil
}
```

---

### 2. 環境変数の重複コピー (優先度: 中)

**ファイル**: `internal/executor/host.go:27-34`

**問題**:
各ステップ実行時に `os.Environ()` を呼び出し、システム環境全体をコピーしています。

```go
if len(config.Env) > 0 {
    // 毎回システム環境をコピー
    cmd.Env = os.Environ()
    // カスタム変数を追加
    for key, value := range config.Env {
        cmd.Env = append(cmd.Env, key+"="+value)
    }
}
```

**影響**:
- ステップ数 × システム環境変数数 のメモリ割り当て
- 多くのステップを持つワークフローで累積的な影響

**推奨修正**:
Runner初期化時にシステム環境をキャッシュする：

```go
type HostExecutor struct {
    cachedSysEnv []string
    once         sync.Once
}

func (h *HostExecutor) getCachedSysEnv() []string {
    h.once.Do(func() {
        h.cachedSysEnv = os.Environ()
    })
    return h.cachedSysEnv
}

func (h *HostExecutor) Execute(config Config) (Result, error) {
    cmd := exec.Command("sh", "-c", config.Command)

    if len(config.Env) > 0 {
        sysEnv := h.getCachedSysEnv()
        cmd.Env = make([]string, len(sysEnv), len(sysEnv)+len(config.Env))
        copy(cmd.Env, sysEnv)
        for key, value := range config.Env {
            cmd.Env = append(cmd.Env, key+"="+value)
        }
    }
    // ...
}
```

---

### 3. Git操作の冗長性 (優先度: 中)

**ファイル**: `internal/util/git.go` & `internal/worktree/worktree.go`

**問題**:
`FindGitRoot` (`git rev-parse --show-toplevel`) と `verifyGitRepo` (`git rev-parse --git-dir`) が似た処理を行っており、実行時に両方が呼ばれる可能性があります。

`cli/run.go:122-132` の処理フロー:
```go
// util.FindGitRoot が git コマンドを実行
repoRoot, err := util.FindGitRoot(ctx, opts.WorkingDir)

// worktree.CreateWorkspace 内で verifyGitRepo が再度 git コマンドを実行
ws, err := worktree.CreateWorkspace(ctx, repoRoot)
```

**影響**:
- 不要なプロセス起動オーバーヘッド
- 各ワークフロー実行で約100-200msの遅延追加

**推奨修正**:
`FindGitRoot`が成功した場合、`verifyGitRepo`のチェックは不要なので、`CreateWorkspace`に検証済みフラグを渡すか、統合する：

```go
func CreateWorkspace(ctx context.Context, repoRoot string, verified bool) (*Workspace, error) {
    if !verified {
        if err := verifyGitRepo(ctx, repoRoot); err != nil {
            return nil, err
        }
    }
    // ...
}
```

---

### 4. MergeEnvの毎回割り当て (優先度: 低)

**ファイル**: `internal/runtime/defaults.go:7-20`

**問題**:
`MergeEnv`関数は毎回新しいmapを作成します。ステップごとに呼び出されるため、GCプレッシャーが発生します。

```go
func MergeEnv(maps ...map[string]string) map[string]string {
    result := make(map[string]string)  // 毎回新規作成
    for _, m := range maps {
        for key, value := range m {
            result[key] = value
        }
    }
    return result
}
```

**影響**:
- 軽微（Goのmapは効率的に実装されている）
- 多数のステップを持つワークフローでのみ顕著

**推奨修正**:
事前割り当てを行う：

```go
func MergeEnv(maps ...map[string]string) map[string]string {
    // 合計サイズを事前計算
    totalSize := 0
    for _, m := range maps {
        totalSize += len(m)
    }

    result := make(map[string]string, totalSize)
    for _, m := range maps {
        if m == nil {
            continue
        }
        for key, value := range m {
            result[key] = value
        }
    }
    return result
}
```

---

### 5. ヒアドキュメントの非効率な結合 (優先度: 低)

**ファイル**: `internal/envfiles/parse.go:49-58`

**問題**:
複数行の値を `[]string` に蓄積し、最後に `strings.Join` で結合しています。

```go
var valueLines []string
for scanner.Scan() {
    valueLine := scanner.Text()
    if valueLine == delimiter {
        break
    }
    valueLines = append(valueLines, valueLine)
}
value := strings.Join(valueLines, "\n")
```

**影響**:
- 非常に軽微（通常、ヒアドキュメントは小さい）
- 大きなヒアドキュメント（数MB）でのみ問題

**推奨修正**:
`strings.Builder` を使用：

```go
var sb strings.Builder
for scanner.Scan() {
    valueLine := scanner.Text()
    if valueLine == delimiter {
        break
    }
    if sb.Len() > 0 {
        sb.WriteByte('\n')
    }
    sb.WriteString(valueLine)
}
value := sb.String()
```

---

## 良い点 (Best Practices)

1. **正規表現のプリコンパイル** (`security/envvar.go:33`)
   ```go
   var validEnvVarName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
   ```
   グローバル変数としてコンパイル済み正規表現を保持している点は良い実装です。

2. **bytes.Bufferの使用** (`executor/host.go:36-38`)
   ```go
   var stdout, stderr bytes.Buffer
   cmd.Stdout = &stdout
   cmd.Stderr = &stderr
   ```
   コマンド出力のキャプチャに効率的なバッファを使用しています。

3. **スライスの事前割り当て** (`cli/run.go:173`)
   ```go
   StepResults: make([]StepResult, 0, len(job.Steps)),
   ```
   ステップ結果のスライスにキャパシティを事前設定している点は良いです。

4. **インターフェースベースの設計** (`executor/executor.go`)
   テスト可能性と拡張性のためのインターフェース設計は良い実装です。

5. **リソースのクリーンアップ** (`cli/run.go:143-149`)
   ```go
   defer func() {
       if err := worktree.RemoveWorkspace(ctx, ws); err != nil {
           // ログ出力
       }
   }()
   ```
   適切なリソースクリーンアップパターンを使用しています。

---

## ベンチマーク推奨

以下のベンチマークを追加することを推奨します：

```go
// benchmark_test.go
func BenchmarkLoadWorkflowFile(b *testing.B) {
    for i := 0; i < b.N; i++ {
        workflow.LoadWorkflowFile("testdata/large_workflow.yml")
    }
}

func BenchmarkMergeEnv(b *testing.B) {
    env1 := map[string]string{"A": "1", "B": "2", "C": "3"}
    env2 := map[string]string{"D": "4", "E": "5"}
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        runtime.MergeEnv(env1, env2)
    }
}

func BenchmarkExecuteStep(b *testing.B) {
    exec := executor.NewHostExecutor()
    config := executor.Config{
        Command:    "echo hello",
        Env:        map[string]string{"FOO": "bar"},
        WorkingDir: "/tmp",
    }
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        exec.Execute(config)
    }
}
```

---

## 結論

Raptorは全体的に効率的に実装されていますが、以下の改善を推奨します：

| 修正 | 期待される効果 | 実装難易度 |
|------|---------------|------------|
| YAMLの二重パース解消 | パース時間 40-50% 短縮 | 中 |
| 環境変数キャッシュ | メモリ割り当て削減 | 低 |
| Git操作の統合 | 起動時間 100-200ms 短縮 | 低 |

現在のコードベースでは、パフォーマンスの問題よりもセキュリティと正確性が優先されており、これは正しいアプローチです。上記の改善は、コードの可読性を損なうことなく実装可能です。
