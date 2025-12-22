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

#### Linux / macOS

```bash
# バイナリをダウンロード（例: Linux x86_64）
curl -LO https://github.com/watany-dev/raptor/releases/latest/download/raptor_<VERSION>_Linux_x86_64.tar.gz

# 解凍
tar xzf raptor_<VERSION>_Linux_x86_64.tar.gz

# 実行可能パスに移動
sudo mv raptor /usr/local/bin/

# 動作確認
raptor --version
```

#### Windows

1. [リリースページ](https://github.com/watany-dev/raptor/releases)から `raptor_<VERSION>_Windows_x86_64.zip` をダウンロード
2. ZIPファイルを解凍
3. `raptor.exe` をPATHの通ったディレクトリに配置

### Go installを使用

```bash
go install github.com/watany-dev/raptor/cmd/raptor@latest
```

### ソースからビルド

```bash
# リポジトリのクローン
git clone https://github.com/watany-dev/raptor.git
cd raptor

# ビルド
go build -o raptor ./cmd/raptor

# パスに追加 (オプション)
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
| `uses` (アクション) | ❌ |
| `with` (アクション入力) | ❌ |
| `if` (条件分岐) | ❌ |
| `matrix` (マトリックスビルド) | ❌ |

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
│   ├── util/          # Git操作ユーティリティ
│   ├── workflow/      # ワークフローYAML解析
│   └── worktree/      # Git worktree管理
├── docs/              # 開発ドキュメント
└── testdata/          # テスト用ワークフロー
```

## ライセンス

Apache License 2.0
