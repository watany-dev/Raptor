# Development Plans

このディレクトリには、Raptorの機能開発計画が格納されています。

## ディレクトリ構造

```
plans/
├── README.md                      # このファイル
├── completed/                     # 完了済み開発計画
│   ├── plan-complex-conditions.md # 複雑な条件式のサポート（PR #53で完了）
│   └── plan-tdd-development.md    # TDD開発計画（全フェーズ完了）
└── plan-job-dependency-graph.md   # ジョブ依存関係グラフ（needs）の実装計画
```

## 計画文書の管理

### 新規計画の追加

新しい機能の開発計画を作成する場合：

1. `plan-<feature-name>.md` という形式でこのディレクトリに配置
2. 計画には以下の情報を含める：
   - 目標と概要
   - イテレーション別のタスク
   - ファイル構成
   - 完了基準

### 計画の完了

開発が完了した計画は：

1. `completed/` ディレクトリに移動
2. ファイル名を `plan-<feature-name>.md` の形式で統一
3. ドキュメント冒頭に完了日とPR番号を記載

## 現在の開発状況

- ✅ **複雑な条件式サポート** - 完了（PR #53）
- ✅ **TDD開発** - 全フェーズ完了
- 📋 **ジョブ依存関係グラフ（needs）** - 計画段階
