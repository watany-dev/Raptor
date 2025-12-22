# Raptor TDD開発計画

このドキュメントは `docs/impl/overview.md` に基づき、TDD（テスト駆動開発）スタイルでの実装計画を定義します。

---

## 開発方針

### TDDサイクル
各機能は以下のサイクルで実装します:
1. **Red**: テストを書く（失敗する）
2. **Green**: 最小限の実装でテストを通す
3. **Refactor**: コードを改善する

### イテレーション単位
機能を最小単位に分割し、各イテレーションで1つの機能を完成させます。

---

## Phase 1: 基盤構築

### Iteration 1.1: プロジェクト構造とFindGitRoot
**目標**: Gitリポジトリのルートを検出する

**テストケース**:
- [ ] 現在のディレクトリがGitリポジトリの場合、ルートパスを返す
- [ ] サブディレクトリから実行した場合、親のルートパスを返す
- [ ] Gitリポジトリ外から実行した場合、エラーを返す

**ファイル**:
- `internal/util/git.go`
- `internal/util/git_test.go`

### Iteration 1.2: GitHeadSHA / GitHeadRef
**目標**: 現在のHEADのSHAとrefを取得する

**テストケース**:
- [ ] 有効なリポジトリからSHAを取得できる
- [ ] 有効なリポジトリからrefを取得できる
- [ ] 無効なリポジトリではエラーを返す

**ファイル**:
- `internal/util/git.go`
- `internal/util/git_test.go`

---

## Phase 2: Worktree管理

### Iteration 2.1: CreateWorkspace
**目標**: 隔離されたワークスペースを作成する

**テストケース**:
- [x] 新しいworktreeを作成できる
- [x] ワークスペースIDがユニークである
- [x] 作成されたworktreeが正しいパスにある

**ファイル**:
- `internal/worktree/types.go`
- `internal/worktree/worktree.go`
- `internal/worktree/worktree_test.go`

### Iteration 2.2: RemoveWorkspace
**目標**: ワークスペースをクリーンアップする

**テストケース**:
- [ ] worktreeを正常に削除できる
- [ ] 削除後、ディレクトリが存在しない

**ファイル**:
- `internal/worktree/worktree.go`
- `internal/worktree/worktree_test.go`

---

## Phase 3: ワークフロー処理

### Iteration 3.1: WorkflowFile型定義とYAMLロード
**目標**: ワークフローYAMLをパースする

**テストケース**:
- [ ] 有効なYAMLをパースできる
- [ ] name, env, jobsが正しく読み込まれる
- [ ] 無効なYAMLでエラーを返す

**ファイル**:
- `internal/workflow/model.go`
- `internal/workflow/load.go`
- `internal/workflow/load_test.go`

### Iteration 3.2: ワークフロー探索
**目標**: .github/workflows/*.ymlを発見する

**テストケース**:
- [ ] workflowsディレクトリからYAMLファイルを発見できる
- [ ] workflowsディレクトリがない場合エラーを返す

**ファイル**:
- `internal/workflow/select.go`
- `internal/workflow/select_test.go`

### Iteration 3.3: Job選択
**目標**: 特定のJobを選択する

**テストケース**:
- [ ] 存在するJobIDで正しいJobを返す
- [ ] 存在しないJobIDでエラーを返す

**ファイル**:
- `internal/workflow/select.go`
- `internal/workflow/select_test.go`

---

## Phase 4: 実行エンジン

### Iteration 4.1: HostExecutor基本実装
**目標**: シェルコマンドを実行する

**テストケース**:
- [ ] 単純なコマンドを実行できる
- [ ] 終了コードを正しく返す
- [ ] 環境変数が正しく設定される

**ファイル**:
- `internal/executor/executor.go`
- `internal/executor/host.go`
- `internal/executor/host_test.go`

### Iteration 4.2: 環境変数マージ
**目標**: workflow → job → step の順でenvをマージ

**テストケース**:
- [ ] 複数のmapが正しくマージされる
- [ ] 後のmapが前のmapを上書きする

**ファイル**:
- `internal/runtime/defaults.go`
- `internal/runtime/defaults_test.go`

---

## Phase 5: 環境ファイル

### Iteration 5.1: GITHUB_ENVパース
**目標**: GITHUB_ENVファイルをパースして環境変数に反映

**テストケース**:
- [ ] KEY=VALUE形式をパースできる
- [ ] マルチラインデリミタ形式をパースできる

**ファイル**:
- `internal/envfiles/parse.go`
- `internal/envfiles/parse_test.go`

### Iteration 5.2: GITHUB_PATHパース
**目標**: GITHUB_PATHファイルをパースしてPATHに追加

**テストケース**:
- [ ] 複数行のパスを読み込める
- [ ] PATHの先頭に追加される

**ファイル**:
- `internal/envfiles/parse.go`
- `internal/envfiles/parse_test.go`

---

## Phase 6: CLI統合

### Iteration 6.1: 基本CLI
**目標**: `raptor run`コマンドの基本形

**テストケース**:
- [ ] --workflow フラグを受け付ける
- [ ] --job フラグを受け付ける
- [ ] ヘルプが表示される

**ファイル**:
- `cmd/raptor/main.go`
- `internal/cli/flags.go`
- `internal/cli/run.go`

### Iteration 6.2: ジョブ実行ループ
**目標**: 全ステップを順番に実行

**テストケース**:
- [ ] 複数のステップが順番に実行される
- [ ] ステップ間で環境変数が引き継がれる

**ファイル**:
- `internal/cli/run.go`
- `internal/cli/run_test.go`

---

## 現在の進捗

| Phase | Iteration | 状態 |
|-------|-----------|------|
| 1 | 1.1 FindGitRoot | ✅ 完了 |
| 1 | 1.2 GitHeadSHA/Ref | ✅ 完了 |
| 2 | 2.1 CreateWorkspace | ✅ 完了 |
| 2 | 2.2 RemoveWorkspace | 🔄 次に実装 |
| 3 | 3.1 WorkflowFile | ⏳ 待機 |
| 3 | 3.2 ワークフロー探索 | ⏳ 待機 |
| 3 | 3.3 Job選択 | ⏳ 待機 |
| 4 | 4.1 HostExecutor | ⏳ 待機 |
| 4 | 4.2 環境変数マージ | ⏳ 待機 |
| 5 | 5.1 GITHUB_ENV | ⏳ 待機 |
| 5 | 5.2 GITHUB_PATH | ⏳ 待機 |
| 6 | 6.1 基本CLI | ⏳ 待機 |
| 6 | 6.2 ジョブ実行ループ | ⏳ 待機 |

---

## 次のアクション

**Iteration 2.2: RemoveWorkspace** を実装

1. `internal/worktree/worktree_test.go` にRemoveWorkspaceのテストを追加
2. テストが失敗することを確認
3. `internal/worktree/worktree.go` にRemoveWorkspaceを実装（既に実装済み）
4. テストが成功することを確認

> Note: RemoveWorkspaceは2.1で既に実装されているため、テスト追加のみで完了する可能性があります。
