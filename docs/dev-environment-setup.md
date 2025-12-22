# 開発環境準備計画

このドキュメントは `docs/impl/overview.md` を元に、Raptor プロジェクトの開発環境準備手順をまとめたものです。

---

## 1. 必須ツール

### 1.1 Go言語環境
- **バージョン**: Go 1.22 以上
- **インストール確認**:
  ```bash
  go version
  # 期待される出力: go version go1.22.x linux/amd64 (または以上)
  ```
- **インストール方法** (未インストールの場合):
  ```bash
  # Ubuntu/Debian
  sudo apt update
  sudo apt install golang-go

  # または公式バイナリを使用
  wget https://go.dev/dl/go1.22.linux-amd64.tar.gz
  sudo tar -C /usr/local -xzf go1.22.linux-amd64.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  ```

### 1.2 Git
- **バージョン**: 2.5+ (git worktree機能が必要)
- **インストール確認**:
  ```bash
  git --version
  git worktree list  # worktree機能の確認
  ```

---

## 2. プロジェクト構造のセットアップ

### 2.1 ディレクトリ構造の作成
以下のディレクトリ構造を作成する必要があります:

```
raptor/
├── cmd/raptor/
│   └── main.go
├── internal/
│   ├── cli/
│   │   ├── flags.go
│   │   └── run.go
│   ├── worktree/
│   │   ├── worktree.go
│   │   └── types.go
│   ├── workflow/
│   │   ├── load.go
│   │   ├── model.go
│   │   ├── select.go
│   │   └── plan.go
│   ├── runtime/
│   │   ├── defaults.go
│   │   ├── interpolate.go
│   │   └── state.go
│   ├── envfiles/
│   │   ├── envfiles.go
│   │   ├── parse.go
│   │   └── types.go
│   ├── executor/
│   │   ├── executor.go
│   │   └── host.go
│   └── util/
│       ├── git.go
│       ├── env.go
│       └── fs.go
├── testdata/           # テスト用ワークフローYAML
├── go.mod
├── go.sum
└── README.md
```

### 2.2 セットアップスクリプト
```bash
# プロジェクトルートで実行
mkdir -p raptor/cmd/raptor
mkdir -p raptor/internal/{cli,worktree,workflow,runtime,envfiles,executor,util}
mkdir -p raptor/testdata
```

---

## 3. Go モジュールの初期化

### 3.1 go.mod の作成
```bash
cd raptor
go mod init github.com/watany-dev/Raptor/raptor
```

### 3.2 必須依存パッケージ
| パッケージ | バージョン | 用途 |
|-----------|-----------|------|
| `gopkg.in/yaml.v3` | v3.0.1 | YAML パース |

```bash
go get gopkg.in/yaml.v3@v3.0.1
```

### 3.3 オプション依存パッケージ (MVP後)
| パッケージ | 用途 |
|-----------|------|
| `github.com/spf13/cobra` | CLIフレームワーク |
| `log/slog` (stdlib) | 構造化ログ |

---

## 4. 環境変数の設定

開発時に便利な環境変数:
```bash
# .envrc または .bashrc に追加
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
export GO111MODULE=on
```

---

## 5. 開発ツール (推奨)

### 5.1 コード品質ツール
```bash
# gofmt (標準付属)
gofmt -w .

# golint
go install golang.org/x/lint/golint@latest

# staticcheck
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### 5.2 テストツール
```bash
# 標準テスト実行
go test ./...

# カバレッジ
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 6. セットアップ確認チェックリスト

- [ ] Go 1.22+ がインストールされている
- [ ] Git 2.5+ がインストールされている (`git worktree` が使える)
- [ ] プロジェクトディレクトリ構造が作成されている
- [ ] `go mod init` が完了している
- [ ] `gopkg.in/yaml.v3` が追加されている
- [ ] `go build ./...` がエラーなく完了する
- [ ] `go test ./...` が実行できる

---

## 7. 実装順序 (参考)

overview.md で定義された実装順序:

1. **CLI + リポジトリ検出のスキャフォールド**
   - `raptor run --workflow --job`
   - `FindGitRoot`

2. **Worktreeワークスペースライフサイクル**
   - `CreateWorkspace` / `RemoveWorkspace`

3. **ワークフロー探索 + YAMLローダー**
   - `.github/workflows/*.yml` の探索
   - YAML を `WorkflowFile` にロード

4. **Job選択 + ステップ計画**
   - Job選択、ステップのフラット化
   - ステップごとの shell/working-directory 解決

5. **HostExecutor (run ステップ用)**
   - 正しい Dir と env で run を実行
   - 終了コードとエラーの明確な処理

6. **環境レイヤリング**
   - `MergeEnv` 実装と plan への適用

7. **環境ファイル**
   - ステップごとの temp dir 作成
   - GITHUB_ENV, GITHUB_PATH, GITHUB_OUTPUT, GITHUB_STEP_SUMMARY の設定
   - ステップ後: ランタイム状態への parse/apply

8. **最小デフォルト環境変数**
   - GITHUB_WORKSPACE, GITHUB_ACTIONS, CI, GITHUB_SHA, GITHUB_REF

9. **オプション: 最小 ${{ }} 補間**

10. **テスト + 堅牢化**
    - envfiles と env merge の単体テスト
    - worktree + ステップチェーンの統合テスト

---

## 8. クイックスタート

```bash
# 1. リポジトリのクローン
git clone <repository-url>
cd Raptor

# 2. 開発環境の確認
go version        # 1.22+ を確認
git --version     # 2.5+ を確認

# 3. プロジェクト構造の作成
mkdir -p raptor/cmd/raptor
mkdir -p raptor/internal/{cli,worktree,workflow,runtime,envfiles,executor,util}

# 4. Go モジュールの初期化
cd raptor
go mod init github.com/watany-dev/Raptor/raptor
go get gopkg.in/yaml.v3@v3.0.1

# 5. ビルド確認 (初期ファイル作成後)
go build ./...
```

---

## 備考

- MVP では `act` に依存せず、独自実装で GitHub Actions のサブセットを実行
- Docker Executor は将来の拡張として設計に含まれているが、MVP では HostExecutor のみ
- `uses:` アクションはMVPのスコープ外
