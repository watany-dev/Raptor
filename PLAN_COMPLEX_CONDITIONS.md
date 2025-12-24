# 複雑な条件式対応 開発計画

## ✅ 実装完了 (2024年12月24日)

すべてのフェーズが実装完了しました。

### ライブラリ移行 (2024年12月24日)

独自実装のトークナイザー/パーサー/ASTを `expr-lang/expr` ライブラリに移行しました。

**削除されたファイル**:
- `tokenizer.go`, `tokenizer_test.go`
- `parser.go`, `parser_test.go`
- `ast.go`

**変更点**:
- `evaluator.go` を `expr-lang/expr` を使用するように書き換え
- コード行数: ~850行 → ~220行 (約75%削減)
- 全テストが継続してパス

## 目標
`internal/expression/evaluator.go` を拡張し、以下の構文に対応する：
- AND/OR演算子 (`&&`, `||`) ✅
- 否定演算子 (`!`) ✅
- 関数: `contains()`, `startsWith()`, `endsWith()`, `hashFiles()` ✅

## 現状分析

### サポートしている構文
- リテラル: `true`, `false`
- ステータス関数: `always()`, `success()`, `failure()`, `cancelled()`
- 環境変数比較: `env.VAR == 'value'`, `env.VAR != 'value'`
- ステップ結果: `steps.ID.outcome == 'value'`
- **NEW** 論理演算子: `&&`, `||`, `!`
- **NEW** 文字列関数: `contains()`, `startsWith()`, `endsWith()`
- **NEW** ファイルハッシュ: `hashFiles()`
- **NEW** グループ化: `(expression)`

---

## フェーズ別開発計画

### フェーズ 1: トークナイザーの実装 ✅
**目標**: 条件式を意味のある単位（トークン）に分解する

**ファイル**: `internal/expression/tokenizer.go`

**対応するトークン種別**:
```go
type TokenType int

const (
    TOKEN_AND        // &&
    TOKEN_OR         // ||
    TOKEN_NOT        // !
    TOKEN_LPAREN     // (
    TOKEN_RPAREN     // )
    TOKEN_EQ         // ==
    TOKEN_NE         // !=
    TOKEN_STRING     // 'value'
    TOKEN_IDENT      // env.VAR, steps.id.outcome, function names
    TOKEN_COMMA      // ,
    TOKEN_EOF
)
```

**タスク**:
1. [x] `Token` 構造体を定義
2. [x] `Tokenizer` 構造体を実装
3. [x] `NextToken()` メソッドを実装
4. [x] トークナイザーのユニットテスト作成 (`tokenizer_test.go`)

---

### フェーズ 2: パーサー/AST の実装 ✅
**目標**: トークン列を抽象構文木（AST）に変換する

**ファイル**: `internal/expression/parser.go`, `internal/expression/ast.go`

**タスク**:
1. [x] AST ノード型を定義
2. [x] `Parser` 構造体を実装
3. [x] 再帰下降パーサーを実装
   - `parseExpression()` - OR を処理
   - `parseAndExpr()` - AND を処理
   - `parseUnaryExpr()` - NOT を処理
   - `parseComparisonExpr()` - ==, != を処理
   - `parsePrimaryExpr()` - リテラル、関数、識別子を処理
4. [x] パーサーのユニットテスト作成 (`parser_test.go`)

---

### フェーズ 3: AST評価器の実装 ✅
**目標**: ASTを走査して真偽値を返す評価器を実装

**ファイル**: `internal/expression/evaluator.go` (既存ファイルを拡張)

**タスク**:
1. [x] `EvaluationContext` 構造体を定義
2. [x] `evaluateNode(node Node, ctx *EvaluationContext) (interface{}, error)` を実装
3. [x] BinaryExpr の評価（`&&`, `||`, `==`, `!=`）
4. [x] UnaryExpr の評価（`!`）
5. [x] 既存のステータス関数を CallExpr として評価
6. [x] 識別子の解決（`env.VAR`, `steps.id.outcome`）
7. [x] 既存の `Evaluate()` メソッドを新しい実装に切り替え
8. [x] 後方互換性のテスト（既存テストが全て通ること）

---

### フェーズ 4: 論理演算子 (`&&`, `||`, `!`) の統合テスト ✅
**目標**: 論理演算子が正しく動作することを確認

**ファイル**: `internal/expression/evaluator_test.go` を拡張

**実装済みテストケース**:
- AND演算子: `success() && true`, `success() && failure()`, etc.
- OR演算子: `success() || failure()`, `failure() || true`, etc.
- NOT演算子: `!failure()`, `!success()`, etc.
- 複合条件: `success() && !failure()`, `(success() || failure()) && true`
- 短絡評価: `false && unknown_func()`, `true || unknown_func()`

---

### フェーズ 5: `contains()` 関数の実装 ✅
**目標**: 文字列や配列に値が含まれるかチェック

**タスク**:
1. [x] `contains()` を CallExpr 評価に追加
2. [x] 大文字/小文字を無視した比較を実装
3. [x] ユニットテスト作成

---

### フェーズ 6: `startsWith()` 関数の実装 ✅
**目標**: 文字列が特定のプレフィックスで始まるかチェック

**タスク**:
1. [x] `startsWith()` を CallExpr 評価に追加
2. [x] 大文字/小文字を無視した比較を実装
3. [x] ユニットテスト作成

---

### フェーズ 7: `endsWith()` 関数の実装 ✅
**目標**: 文字列が特定のサフィックスで終わるかチェック

**タスク**:
1. [x] `endsWith()` を CallExpr 評価に追加
2. [x] ユニットテスト作成

---

### フェーズ 8: `hashFiles()` 関数の実装 ✅
**目標**: ファイルのハッシュ値を計算してキャッシュキーなどに使用

**タスク**:
1. [x] `hashFiles()` を CallExpr 評価に追加
2. [x] glob パターンマッチングを実装
3. [x] SHA256 ハッシュ計算を実装
4. [x] ファイルが見つからない場合は空文字を返す
5. [x] ユニットテスト作成

---

### フェーズ 9: 既存 API との統合と最終テスト ✅
**目標**: 新実装を既存のワークフロー実行に統合

**タスク**:
1. [x] `ConditionEvaluator.Evaluate()` を新パーサー/評価器に切り替え
2. [x] 古い正規表現ロジックを削除
3. [x] `step_executor.go` との統合テスト
4. [x] 全プロジェクトテストがパス

---

### フェーズ 10: ドキュメントとクリーンアップ ✅
**目標**: コードの品質とドキュメントを整備

**タスク**:
1. [x] GoDoc コメントを全ての公開APIに追加
2. [x] このドキュメントを更新
3. [x] 未使用コードの削除
4. [x] `go vet` を通す

---

## ファイル構成（最終形）

```
internal/expression/
├── tokenizer.go       # フェーズ1: トークナイザー
├── tokenizer_test.go
├── ast.go             # フェーズ2: AST定義
├── parser.go          # フェーズ2: パーサー
├── parser_test.go
├── evaluator.go       # フェーズ3-8: 評価器（組み込み関数含む）
└── evaluator_test.go  # 拡張済み
```

---

## 参考リンク
- [GitHub Actions 式の構文](https://docs.github.com/en/actions/learn-github-actions/expressions)
- [ステータス関数](https://docs.github.com/en/actions/learn-github-actions/expressions#status-check-functions)
