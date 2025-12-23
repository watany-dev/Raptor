# テストカバレッジ改善計画

## 現状 (2025-12-23時点)

| パッケージ | 現在のカバレッジ | 目標 |
|------------|-----------------|------|
| `cmd/raptor` | 0.0% | 80% → 100% |
| `internal/cli` | 67.9% | 80% → 100% |
| `internal/envfiles` | 92.3% | 100% |
| `internal/executor` | 95.7% | 100% |
| `internal/expression` | 100.0% | ✓ 完了 |
| `internal/runtime` | 100.0% | ✓ 完了 |
| `internal/security` | 96.2% | 100% |
| `internal/util` | 95.7% | 100% |
| `internal/workflow` | 86.4% | 100% |
| `internal/worktree` | 78.0% | 80% → 100% |

**総合カバレッジ: 75.1% → 目標: 100%**

---

## フェーズ1: 80%未満を80%以上に引き上げる

### 1.1 `internal/cli` (67.9% → 80%)

優先度: **高**

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `dry_run.go` 全体 | 0.0% | 新規テストファイル作成 |
| `updateEnvironmentFromFiles` | 35.0% | エラーケースのテスト追加 |
| `Execute` | 75.0% | 境界ケースのテスト追加 |
| `printOutput` | 75.0% | 出力パターンのテスト追加 |

- [ ] `dry_run_test.go` を新規作成
- [ ] `step_executor_test.go` のテストケース追加

### 1.2 `internal/worktree` (78.0% → 80%)

優先度: **高**

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `RemoveWorkspace` | 61.5% | エラーハンドリングのテスト |
| `generateID` | 75.0% | エッジケースのテスト |

- [ ] `worktree_test.go` にエラーケースを追加

---

## フェーズ2: 80%以上を100%に引き上げる (優先度順)

### 2.1 `internal/workflow` (86.4% → 100%)

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `extractJobOrderFromNode` | 80.0% | YAMLパースエラーのテスト |
| `DiscoverWorkflows` | 80.0% | ファイル探索エラーのテスト |
| `LoadWorkflowFile` | 90.9% | 残りのエラーパスをカバー |

- [ ] `load_test.go` にエラーケース追加
- [ ] `select_test.go` にエラーケース追加

### 2.2 `internal/envfiles` (92.3% → 100%)

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `ParseEnvFile` | 92.9% | 残りのパースエラーをカバー |
| `ParsePathFile` | 88.2% | エッジケースのテスト |

- [ ] `parse_test.go` にエッジケース追加

### 2.3 `internal/executor` (95.7% → 100%)

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `Execute` | 94.7% | 残りのエラーパスをカバー |

- [ ] `host_test.go` にエラーケース追加

### 2.4 `internal/util` (95.7% → 100%)

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `GitHeadRef` | 85.7% | gitコマンドエラーのテスト |

- [ ] `git_test.go` にエラーケース追加

### 2.5 `internal/security` (96.2% → 100%)

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `ValidateWorkingDirectory` | 91.7% | 残りのバリデーションエラーをカバー |

- [ ] `path_test.go` にエッジケース追加

### 2.6 `internal/worktree` (80% → 100%)

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `CreateWorkspace` | 82.4% | ワークスペース作成エラーのテスト |
| `RemoveWorkspace` | 61.5% | 削除エラーのテスト |
| `generateID` | 75.0% | ID生成のエッジケース |

- [ ] `worktree_test.go` を大幅に拡充

### 2.7 `internal/cli` (80% → 100%)

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `Run` | 85.7% | 実行フローのテスト |
| `setupRunContext` | 86.7% | コンテキスト設定エラーのテスト |
| `runJob` | 96.2% | ジョブ実行エラーのテスト |
| `ParseRunFlags` | 90.5% | フラグパースエラーのテスト |
| `handleSkippedStep` | 83.3% | スキップ条件のテスト |
| `executeStep` | 85.0% | ステップ実行エラーのテスト |

- [ ] 全テストファイルの網羅的なテストケース追加

---

## フェーズ3: cmd/raptor のテスト (0% → 80% → 100%)

優先度: **低** (エントリーポイントのため)

| 関数 | 現在 | 対応内容 |
|------|------|----------|
| `main` | 0.0% | 統合テストで対応 |
| `run` | 0.0% | コマンドライン引数のテスト |
| `runCommand` | 0.0% | サブコマンド実行のテスト |

- [ ] `main_test.go` を新規作成
- [ ] 統合テストの検討

---

## 実施スケジュール

```
フェーズ1: 80%未満 → 80%以上
├── 1.1 internal/cli (dry_run.go, step_executor.go)
└── 1.2 internal/worktree

フェーズ2: 80%以上 → 100% (優先度順)
├── 2.1 internal/workflow
├── 2.2 internal/envfiles
├── 2.3 internal/executor
├── 2.4 internal/util
├── 2.5 internal/security
├── 2.6 internal/worktree
└── 2.7 internal/cli

フェーズ3: cmd/raptor
└── 3.1 main.go のテスト追加
```

---

## 進捗トラッキング

### フェーズ1
- [ ] `internal/cli` 80%達成
- [ ] `internal/worktree` 80%達成

### フェーズ2
- [ ] `internal/workflow` 100%達成
- [ ] `internal/envfiles` 100%達成
- [ ] `internal/executor` 100%達成
- [ ] `internal/util` 100%達成
- [ ] `internal/security` 100%達成
- [ ] `internal/worktree` 100%達成
- [ ] `internal/cli` 100%達成

### フェーズ3
- [ ] `cmd/raptor` 80%達成
- [ ] `cmd/raptor` 100%達成

---

## 注意事項

1. **テストの品質**: カバレッジだけでなく、意味のあるアサーションを書く
2. **エッジケース**: 正常系だけでなく、エラーケースも網羅する
3. **リファクタリング**: テストしにくいコードは適宜リファクタリングを検討
4. **CI/CD**: カバレッジ閾値をCIに設定して退行を防ぐ
