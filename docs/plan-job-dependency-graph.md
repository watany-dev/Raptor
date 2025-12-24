# Job Dependency Graph (`needs`) 実装計画

## 概要

GitHub Actions の `needs` キーワードによるジョブ依存関係機能を Raptor に実装します。これにより、ジョブ間の依存関係を定義し、トポロジカルソートに基づいた正しい実行順序を決定できるようになります。

---

## 📋 イテレーション計画

### **イテレーション 1: データモデルの拡張**
**ファイル:** `internal/workflow/model.go`

**タスク 1.1: Job構造体に Needs フィールドを追加**
```go
type Job struct {
    Name   string            `yaml:"name"`
    RunsOn string            `yaml:"runs-on"`
    Needs  StringOrSlice     `yaml:"needs"`  // 追加
    Env    map[string]string `yaml:"env"`
    Steps  []Step            `yaml:"steps"`
}
```

**タスク 1.2: StringOrSlice カスタム型の実装**
- GitHub Actions では `needs` は文字列または文字列配列を受け付ける
- カスタムUnmarshalYAML実装で両方を処理

```go
// StringOrSlice は単一文字列またはスライスを処理
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalYAML(value *yaml.Node) error {
    // 単一文字列の場合
    if value.Kind == yaml.ScalarNode {
        *s = []string{value.Value}
        return nil
    }
    // 配列の場合
    var slice []string
    if err := value.Decode(&slice); err != nil {
        return err
    }
    *s = slice
    return nil
}
```

**成果物:**
- [ ] `Needs` フィールドがYAMLから正しくパースされる
- [ ] 単一文字列 (`needs: build`) と配列 (`needs: [build, test]`) の両方をサポート

---

### **イテレーション 2: 依存グラフ (DAG) パッケージの作成**
**新規ファイル:** `internal/dag/dag.go`

**タスク 2.1: DAG データ構造の定義**
```go
package dag

// Graph はジョブ間の依存関係を表す有向非巡回グラフ
type Graph struct {
    nodes map[string]bool           // ノード（ジョブID）の集合
    edges map[string][]string       // 隣接リスト（依存先 -> 依存元）
    deps  map[string][]string       // 各ノードの依存関係
}

// New は新しい空のグラフを作成
func New() *Graph

// AddNode はノードを追加
func (g *Graph) AddNode(id string)

// AddEdge は依存関係を追加（from が to に依存）
func (g *Graph) AddEdge(from, to string) error
```

**タスク 2.2: 循環依存検出の実装**
```go
// HasCycle は循環依存があるかチェック
func (g *Graph) HasCycle() bool

// FindCycle は循環依存のパスを返す（デバッグ用）
func (g *Graph) FindCycle() []string
```

**タスク 2.3: トポロジカルソートの実装 (Kahn's Algorithm)**
```go
// TopologicalSort はトポロジカル順序でノードを返す
// 循環がある場合はエラーを返す
func (g *Graph) TopologicalSort() ([]string, error)
```

**アルゴリズム（Kahn's Algorithm）:**
1. 各ノードの入次数（依存元の数）を計算
2. 入次数が0のノードをキューに追加
3. キューからノードを取り出し、結果に追加
4. そのノードの依存先の入次数を減らす
5. 入次数が0になったノードをキューに追加
6. 全ノード処理されなければ循環あり

**成果物:**
- [ ] `dag.Graph` 構造体
- [ ] `TopologicalSort()` メソッド
- [ ] `HasCycle()` / `FindCycle()` メソッド
- [ ] 単体テスト (`internal/dag/dag_test.go`)

---

### **イテレーション 3: ワークフローから依存グラフを構築**
**新規ファイル:** `internal/dag/builder.go`

**タスク 3.1: ワークフローからグラフを構築**
```go
// BuildFromWorkflow はWorkflowFileから依存グラフを構築
func BuildFromWorkflow(wf *workflow.WorkflowFile) (*Graph, error) {
    g := New()

    // 全ジョブをノードとして追加
    for jobID := range wf.Jobs {
        g.AddNode(jobID)
    }

    // 依存関係をエッジとして追加
    for jobID, job := range wf.Jobs {
        for _, dep := range job.Needs {
            if err := g.AddEdge(jobID, dep); err != nil {
                return nil, err
            }
        }
    }

    return g, nil
}
```

**タスク 3.2: 依存関係のバリデーション**
```go
// Validate はグラフの妥当性を検証
// - 存在しないジョブへの依存
// - 循環依存
// - 自己依存
func (g *Graph) Validate() error
```

**エラーケース:**
1. `job 'deploy' depends on unknown job 'buildx'`
2. `circular dependency detected: build -> test -> deploy -> build`
3. `job 'build' cannot depend on itself`

**成果物:**
- [ ] `BuildFromWorkflow()` 関数
- [ ] 包括的なバリデーション
- [ ] 明確なエラーメッセージ

---

### **イテレーション 4: 実行順序の決定ロジックを更新**
**ファイル:** `internal/cli/run.go`

**タスク 4.1: determineJobIDs の更新**
```go
func (r *Runner) determineJobIDs(wf *workflow.WorkflowFile, opts *RunOptions) ([]string, error) {
    if opts.Job != "" {
        // 特定ジョブ指定時は依存関係も含める
        return r.resolveJobWithDependencies(wf, opts.Job)
    }

    // 全ジョブ実行時はトポロジカルソート順
    graph, err := dag.BuildFromWorkflow(wf)
    if err != nil {
        return nil, err
    }

    return graph.TopologicalSort()
}
```

**タスク 4.2: 特定ジョブ実行時の依存解決**
```go
// resolveJobWithDependencies は指定ジョブとその全依存を解決
func (r *Runner) resolveJobWithDependencies(wf *workflow.WorkflowFile, jobID string) ([]string, error) {
    // 再帰的に依存関係を収集
    // トポロジカルソートで順序付け
    // 結果を返す
}
```

**成果物:**
- [ ] トポロジカルソート順での実行
- [ ] 特定ジョブ指定時の依存関係自動解決
- [ ] 循環依存時の明確なエラー

---

### **イテレーション 5: 依存ジョブ失敗時のスキップ処理**
**ファイル:** `internal/cli/run.go`

**タスク 5.1: ジョブ実行結果の追跡**
```go
// JobContext はジョブの実行結果を保持
type JobContext struct {
    Result  string            // "success", "failure", "skipped"
    Outputs map[string]string // ジョブの出力（将来の拡張用）
}

// 実行コンテキストに追加
type runContext struct {
    // 既存フィールド...
    jobResults map[string]*JobContext // ジョブ結果を追跡
}
```

**タスク 5.2: executeJobs の更新**
```go
func (r *Runner) executeJobs(wf *workflow.WorkflowFile, jobIDs []string, opts *RunOptions, runCtx *runContext) ([]*RunResult, error) {
    runCtx.jobResults = make(map[string]*JobContext)
    var results []*RunResult

    for _, jobID := range jobIDs {
        job := wf.Jobs[jobID]

        // 依存ジョブの結果をチェック
        if shouldSkip, reason := r.shouldSkipJob(job, runCtx); shouldSkip {
            fmt.Fprintf(r.stdout, "=== Skipping job: %s (%s) ===\n", jobID, reason)
            runCtx.jobResults[jobID] = &JobContext{Result: "skipped"}
            results = append(results, &RunResult{
                JobID:   jobID,
                Success: false,
                Skipped: true,
            })
            continue
        }

        result, err := r.runJob(wf, jobID, opts, runCtx)
        // ...結果を記録
    }
    return results, nil
}
```

**タスク 5.3: スキップ判定ロジック**
```go
// shouldSkipJob は依存ジョブの失敗に基づきスキップすべきか判定
func (r *Runner) shouldSkipJob(job *workflow.Job, runCtx *runContext) (bool, string) {
    for _, depJobID := range job.Needs {
        depResult, exists := runCtx.jobResults[depJobID]
        if !exists {
            return true, fmt.Sprintf("dependency '%s' not executed", depJobID)
        }
        if depResult.Result != "success" {
            return true, fmt.Sprintf("dependency '%s' %s", depJobID, depResult.Result)
        }
    }
    return false, ""
}
```

**成果物:**
- [ ] `RunResult` に `Skipped` フィールド追加
- [ ] 依存失敗時の自動スキップ
- [ ] スキップ理由の明確な出力

---

### **イテレーション 6: Dry-run モードの更新**
**ファイル:** `internal/cli/dryrun.go`

**タスク 6.1: 依存関係の表示**
```go
func (f *DryRunFormatter) Format(wf *workflow.WorkflowFile, jobIDs []string, workflowPath string) ([]*RunResult, error) {
    // ...
    for _, jobID := range jobIDs {
        job := wf.Jobs[jobID]
        fmt.Fprintf(f.w, "  Job: %s\n", jobID)

        // 依存関係を表示
        if len(job.Needs) > 0 {
            fmt.Fprintf(f.w, "    Depends on: %s\n", strings.Join(job.Needs, ", "))
        }
        // ...
    }
}
```

**成果物:**
- [ ] Dry-run出力に依存関係表示

---

### **イテレーション 7: テストの実装**

**タスク 7.1: DAGパッケージのテスト**
**ファイル:** `internal/dag/dag_test.go`

```go
func TestTopologicalSort(t *testing.T) {
    tests := []struct {
        name     string
        edges    [][2]string // from -> to
        want     []string
        wantErr  bool
    }{
        {
            name:  "linear dependency",
            edges: [][2]string{{"deploy", "test"}, {"test", "build"}},
            want:  []string{"build", "test", "deploy"},
        },
        {
            name:    "circular dependency",
            edges:   [][2]string{{"a", "b"}, {"b", "c"}, {"c", "a"}},
            wantErr: true,
        },
        // ...more test cases
    }
}
```

**タスク 7.2: 統合テスト**
**ファイル:** `internal/cli/run_test.go` (拡張)

```go
func TestRunWithNeeds(t *testing.T) {
    // テスト用ワークフロー
    workflowYAML := `
name: test
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo "building"
  test:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - run: echo "testing"
  deploy:
    needs: [build, test]
    runs-on: ubuntu-latest
    steps:
      - run: echo "deploying"
`
    // 実行順序が build -> test -> deploy であることを検証
}
```

**タスク 7.3: エラーケースのテスト**
```go
func TestCircularDependency(t *testing.T) {
    // 循環依存のワークフロー
}

func TestUnknownDependency(t *testing.T) {
    // 存在しないジョブへの依存
}

func TestDependencyFailureSkip(t *testing.T) {
    // 依存ジョブ失敗時のスキップ
}
```

**成果物:**
- [ ] DAGパッケージの100%カバレッジ
- [ ] 統合テスト
- [ ] エッジケースのテスト

---

## 📁 ファイル構成（変更・追加）

```
internal/
├── dag/                    # 新規パッケージ
│   ├── dag.go             # DAG実装
│   ├── dag_test.go        # テスト
│   └── builder.go         # ワークフロー→DAG変換
├── workflow/
│   └── model.go           # Needs フィールド追加
└── cli/
    ├── run.go             # 実行ロジック更新
    └── dryrun.go          # Dry-run表示更新
```

---

## 🎯 実装順序の依存関係

```
イテレーション1 (モデル)
       ↓
イテレーション2 (DAG)
       ↓
イテレーション3 (ビルダー)
       ↓
イテレーション4 (実行順序) ← イテレーション5 (スキップ)
       ↓
イテレーション6 (Dry-run)
       ↓
イテレーション7 (テスト) ← 全イテレーションに並行して実施可能
```

---

## ✅ 完了基準

1. **機能要件**
   - [ ] `needs: job-id` 形式のYAMLパースが動作
   - [ ] `needs: [job1, job2]` 形式のYAMLパースが動作
   - [ ] トポロジカルソートに基づく実行順序
   - [ ] 循環依存の検出とエラー報告
   - [ ] 依存ジョブ失敗時の後続ジョブスキップ
   - [ ] `-j` オプションでの依存関係自動解決

2. **品質要件**
   - [ ] 全テストがパス
   - [ ] 既存のテストに影響なし
   - [ ] エラーメッセージが明確

3. **ドキュメント**
   - [ ] コード内コメント
