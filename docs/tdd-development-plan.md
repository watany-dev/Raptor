# Raptor TDD開発計画

このドキュメントは `docs/impl/overview.md` に基づき、TDD（テスト駆動開発）スタイルでの実装計画を定義します。

---

> **開発方針については [`claude.md`](/claude.md) を参照してください。**

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
- [x] worktreeを正常に削除できる
- [x] 削除後、ディレクトリが存在しない

**ファイル**:
- `internal/worktree/worktree.go`
- `internal/worktree/worktree_test.go`

---

## Phase 3: ワークフロー処理

### Iteration 3.1: WorkflowFile型定義とYAMLロード
**目標**: ワークフローYAMLをパースする

**テストケース**:
- [x] 有効なYAMLをパースできる
- [x] name, env, jobsが正しく読み込まれる
- [x] 無効なYAMLでエラーを返す

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
- [x] 存在するJobIDで正しいJobを返す
- [x] 存在しないJobIDでエラーを返す

**ファイル**:
- `internal/workflow/select.go`
- `internal/workflow/select_test.go`

---

## Phase 4: 実行エンジン

### Iteration 4.1: HostExecutor基本実装
**目標**: シェルコマンドを実行する

**テストケース**:
- [x] 単純なコマンドを実行できる
- [x] 終了コードを正しく返す
- [x] 環境変数が正しく設定される

**ファイル**:
- `internal/executor/executor.go`
- `internal/executor/host.go`
- `internal/executor/host_test.go`

### Iteration 4.2: 環境変数マージ
**目標**: workflow → job → step の順でenvをマージ

**テストケース**:
- [x] 複数のmapが正しくマージされる
- [x] 後のmapが前のmapを上書きする

**ファイル**:
- `internal/runtime/defaults.go`
- `internal/runtime/defaults_test.go`

---

## Phase 5: 環境ファイル

### Iteration 5.1: GITHUB_ENVパース
**目標**: GITHUB_ENVファイルをパースして環境変数に反映

**テストケース**:
- [x] KEY=VALUE形式をパースできる
- [x] マルチラインデリミタ形式をパースできる

**ファイル**:
- `internal/envfiles/parse.go`
- `internal/envfiles/parse_test.go`

### Iteration 5.2: GITHUB_PATHパース
**目標**: GITHUB_PATHファイルをパースしてPATHに追加

**テストケース**:
- [x] 複数行のパスを読み込める
- [x] PATHの先頭に追加される

**ファイル**:
- `internal/envfiles/parse.go`
- `internal/envfiles/parse_test.go`

---

## Phase 6: CLI統合

### Iteration 6.1: 基本CLI
**目標**: `raptor run`コマンドの基本形

**テストケース**:
- [x] --workflow フラグを受け付ける
- [x] --job フラグを受け付ける
- [x] ヘルプが表示される

**ファイル**:
- `cmd/raptor/main.go`
- `internal/cli/flags.go`
- `internal/cli/run.go`

### Iteration 6.2: ジョブ実行ループ
**目標**: 全ステップを順番に実行

**テストケース**:
- [x] 複数のステップが順番に実行される
- [x] ステップ間で環境変数が引き継がれる

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
| 2 | 2.2 RemoveWorkspace | ✅ 完了 |
| 3 | 3.1 WorkflowFile | ✅ 完了 |
| 3 | 3.2 ワークフロー探索 | ✅ 完了 |
| 3 | 3.3 Job選択 | ✅ 完了 |
| 4 | 4.1 HostExecutor | ✅ 完了 |
| 4 | 4.2 環境変数マージ | ✅ 完了 |
| 5 | 5.1 GITHUB_ENV | ✅ 完了 |
| 5 | 5.2 GITHUB_PATH | ✅ 完了 |
| 6 | 6.1 基本CLI | ✅ 完了 |
| 6 | 6.2 ジョブ実行ループ | ✅ 完了 |

---

## 完了

すべてのIterationが完了しました。

### 実装された機能

- `raptor run` コマンド: ワークフローファイルを指定してジョブを実行
- `--workflow` / `-w`: ワークフローファイルのパス指定
- `--job` / `-j`: 実行するジョブID指定
- `--workdir` / `-C`: 作業ディレクトリ指定
- ステップの順次実行
- 環境変数のマージ (workflow → job → step)
- GITHUB_ENV / GITHUB_PATH のサポート

### 使用例

```bash
raptor run --workflow .github/workflows/ci.yml --job build
raptor run -w ci.yml -j test -C /path/to/repo
```
