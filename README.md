# Raptor

GitHub Actions ワークフローをローカルで実行する軽量CLIツール

## 概要

Raptorは、GitHub Actionsのワークフローファイル (`.github/workflows/*.yml`) をローカル環境で実行するためのCLIツールです。CIパイプラインをプッシュ前にローカルでテストできます。

## 特徴

- GitHub Actions ワークフローYAMLのネイティブサポート
- ワークフロー/ジョブ/ステップレベルの環境変数サポート
- `GITHUB_ENV` / `GITHUB_PATH` による動的環境変数の伝搬
- ステップごとの作業ディレクトリ設定
- 軽量でシンプルな設計

## インストール

### バイナリをダウンロード（推奨）

[リリースページ](https://github.com/watany-dev/raptor/releases)から、お使いのプラットフォームに合ったバイナリをダウンロードしてください。

#### Linux (x86_64)

```bash
curl -LO https://github.com/watany-dev/raptor/releases/download/v0.1.0/raptor_0.1.0_Linux_x86_64.tar.gz
tar xzf raptor_0.1.0_Linux_x86_64.tar.gz
sudo mv raptor /usr/local/bin/
raptor --version
```

#### Linux (ARM64)

```bash
curl -LO https://github.com/watany-dev/raptor/releases/download/v0.1.0/raptor_0.1.0_Linux_arm64.tar.gz
tar xzf raptor_0.1.0_Linux_arm64.tar.gz
sudo mv raptor /usr/local/bin/
raptor --version
```

#### macOS (Apple Silicon)

```bash
curl -LO https://github.com/watany-dev/raptor/releases/download/v0.1.0/raptor_0.1.0_Darwin_arm64.tar.gz
tar xzf raptor_0.1.0_Darwin_arm64.tar.gz
sudo mv raptor /usr/local/bin/
raptor --version
```

#### macOS (Intel)

```bash
curl -LO https://github.com/watany-dev/raptor/releases/download/v0.1.0/raptor_0.1.0_Darwin_x86_64.tar.gz
tar xzf raptor_0.1.0_Darwin_x86_64.tar.gz
sudo mv raptor /usr/local/bin/
raptor --version
```

#### Windows (x86_64)

1. [raptor_0.1.0_Windows_x86_64.zip](https://github.com/watany-dev/raptor/releases/download/v0.1.0/raptor_0.1.0_Windows_x86_64.zip) をダウンロード
2. ZIPファイルを解凍
3. `raptor.exe` をPATHの通ったディレクトリに配置

### Go install

```bash
go install github.com/watany-dev/raptor/cmd/raptor@v0.1.0
```

### ソースからビルド

```bash
git clone https://github.com/watany-dev/raptor.git
cd raptor
go build -o raptor ./cmd/raptor
sudo mv raptor /usr/local/bin/
```

### 必要要件

- ランタイム: Git 2.5 以上
- ソースビルド時のみ: Go 1.22 以上

## 使い方

### 基本的な使用方法

```bash
# 特定のジョブを実行
raptor run --workflow <ワークフローファイル> --job <ジョブID>

# 全ジョブを実行 (--job省略時)
raptor run --workflow <ワークフローファイル>
```

### コマンドオプション

| オプション | 短縮形 | 説明 |
|------------|--------|------|
| `--workflow` | `-w` | ワークフローファイルへのパス (必須) |
| `--job` | `-j` | 実行するジョブID (省略時は全ジョブ実行) |
| `--workdir` | `-C` | 作業ディレクトリ (デフォルト: カレントディレクトリ) |

**注意**: すべてのワークフローはセキュリティのため隔離されたGit worktreeで実行されます。

### 使用例

```bash
# CIワークフローのbuildジョブを実行
raptor run --workflow .github/workflows/ci.yml --job build

# 短縮形で指定
raptor run -w ci.yml -j test

# ワークフロー内の全ジョブを実行
raptor run -w ci.yml

# 作業ディレクトリを指定して実行
raptor run -w .github/workflows/ci.yml -j lint -C /path/to/project
```

### ヘルプ

```bash
raptor help
raptor --help
```

### バージョン確認

```bash
raptor version
raptor --version
```

## サポートされる機能

### ワークフロー構文

現在サポートされているGitHub Actions構文:

| 機能 | サポート |
|------|----------|
| `name` (ワークフロー/ジョブ/ステップ名) | ✅ |
| `env` (環境変数) | ✅ |
| `run` (シェルコマンド) | ✅ |
| `working-directory` | ✅ |
| `GITHUB_ENV` | ✅ |
| `GITHUB_PATH` | ✅ |
| `if` (条件分岐) | ✅ (基本構文) |
| `uses` (アクション) | ❌ |
| `with` (アクション入力) | ❌ |
| `matrix` (マトリックスビルド) | ❌ |

### 条件分岐 (`if`)

ステップの条件実行がサポートされています：

```yaml
steps:
  - name: Always run
    if: true
    run: echo "This always runs"

  - name: Conditional
    if: ${{ env.MY_VAR == 'value' }}
    run: echo "Runs when MY_VAR is 'value'"

  - name: On failure
    if: failure()
    run: echo "Runs only if previous step failed"

  - name: Always (even on failure)
    if: always()
    run: echo "Cleanup step"
```

**サポートされる条件構文:**

| 構文 | 説明 |
|------|------|
| `true` / `false` | リテラル真偽値 |
| `success()` | 前のステップがすべて成功 |
| `failure()` | いずれかのステップが失敗 |
| `always()` | 常に実行（失敗後も継続） |
| `${{ env.VAR == 'value' }}` | 環境変数の比較 |
| `${{ env.VAR != 'value' }}` | 環境変数の否定比較 |
| `${{ steps.ID.outcome == 'success' }}` | ステップ結果の参照 |

### サンプルワークフロー

Raptorで実行可能なワークフロー例:

```yaml
name: CI

env:
  GLOBAL_VAR: "global-value"

jobs:
  build:
    runs-on: ubuntu-latest
    env:
      JOB_VAR: "job-value"
    steps:
      - name: Setup
        run: echo "Setting up..."
        env:
          STEP_VAR: "step-value"

      - name: Build
        run: |
          echo "Building..."
          echo "GLOBAL_VAR=$GLOBAL_VAR"

      - name: Set dynamic env
        run: |
          echo "MY_VAR=dynamic-value" >> $GITHUB_ENV

      - name: Use dynamic env
        run: echo "MY_VAR is $MY_VAR"
```

## セキュリティ

Raptorはワークフローファイルに記述されたコマンドを実行するため、**信頼できるワークフローのみを実行してください**。

### セキュリティ機能

- **隔離実行**: すべてのワークフローはGit worktreeで隔離実行されます
- **絶対パス禁止**: `working-directory`で絶対パスは使用できません
- **環境変数保護**: `LD_PRELOAD`等の危険な環境変数はブロックされます
- **入力検証**: 環境変数名と値を検証します

詳細は [SECURITY.md](SECURITY.md) を参照してください。

### 注意事項

Raptorはワークフローを**あなたのユーザー権限で実行**します。悪意のあるワークフローは以下を実行可能です：

- ファイルの削除・変更
- ネットワークアクセス
- 外部へのデータ送信

**必ず内容を確認してから実行してください。**

## 開発

### ビルド

```bash
go build ./...
```

### テスト実行

```bash
go test ./...
```

### テストカバレッジ

```bash
go test -cover ./...
```

## プロジェクト構造

```
raptor/
├── cmd/raptor/        # CLIエントリポイント
├── internal/
│   ├── cli/           # CLIフラグ解析・ランナー
│   ├── envfiles/      # GITHUB_ENV/GITHUB_PATH解析
│   ├── executor/      # コマンド実行エンジン
│   ├── runtime/       # 環境変数マージ処理
│   ├── security/      # セキュリティ検証
│   ├── util/          # Git操作ユーティリティ
│   ├── workflow/      # ワークフローYAML解析
│   └── worktree/      # Git worktree管理
├── docs/              # 開発ドキュメント
└── testdata/          # テスト用ワークフロー
```

## ライセンス

Apache License 2.0
