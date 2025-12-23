# 複雑な条件式対応 開発計画

## 目標
`internal/expression/evaluator.go` を拡張し、以下の構文に対応する：
- AND/OR演算子 (`&&`, `||`)
- 否定演算子 (`!`)
- 関数: `contains()`, `startsWith()`, `hashFiles()`

## 現状分析

### 現在サポートしている構文
- リテラル: `true`, `false`
- ステータス関数: `always()`, `success()`, `failure()`, `cancelled()`
- 環境変数比較: `env.VAR == 'value'`, `env.VAR != 'value'`
- ステップ結果: `steps.ID.outcome == 'value'`

### 課題
現在の実装は正規表現ベースの単純なパターンマッチングで、複合条件をサポートできない。

---

## フェーズ別開発計画

### フェーズ 1: トークナイザーの実装
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
1. [ ] `Token` 構造体を定義
2. [ ] `Tokenizer` 構造体を実装
3. [ ] `NextToken()` メソッドを実装
4. [ ] トークナイザーのユニットテスト作成 (`tokenizer_test.go`)

**テストケース**:
- `success() && failure()` → `[IDENT, AND, IDENT, EOF]`
- `env.VAR == 'value'` → `[IDENT, EQ, STRING, EOF]`
- `!failure()` → `[NOT, IDENT, EOF]`
- `contains(github.event.comment.body, '/deploy')` → `[IDENT, LPAREN, IDENT, COMMA, STRING, RPAREN, EOF]`

---

### フェーズ 2: パーサー/AST の実装
**目標**: トークン列を抽象構文木（AST）に変換する

**ファイル**: `internal/expression/parser.go`

**ASTノード種別**:
```go
type Node interface {
    node()
}

type BinaryExpr struct {  // && || == !=
    Left     Node
    Operator TokenType
    Right    Node
}

type UnaryExpr struct {   // !
    Operator TokenType
    Operand  Node
}

type CallExpr struct {    // contains(), startsWith(), etc.
    FuncName  string
    Arguments []Node
}

type Identifier struct {  // env.VAR, steps.id.outcome
    Value string
}

type StringLiteral struct {
    Value string
}

type BoolLiteral struct {
    Value bool
}
```

**演算子優先順位** (低→高):
1. `||` (論理OR)
2. `&&` (論理AND)
3. `!` (否定)
4. `==`, `!=` (比較)
5. 関数呼び出し、リテラル、識別子

**タスク**:
1. [ ] AST ノード型を定義
2. [ ] `Parser` 構造体を実装
3. [ ] 再帰下降パーサーを実装
   - `parseExpression()` - OR を処理
   - `parseAndExpr()` - AND を処理
   - `parseUnaryExpr()` - NOT を処理
   - `parseComparisonExpr()` - ==, != を処理
   - `parsePrimaryExpr()` - リテラル、関数、識別子を処理
4. [ ] パーサーのユニットテスト作成 (`parser_test.go`)

**テストケース**:
- `true` → `BoolLiteral{true}`
- `success()` → `CallExpr{FuncName: "success"}`
- `a && b` → `BinaryExpr{Left: Ident{a}, Op: AND, Right: Ident{b}}`
- `!failure()` → `UnaryExpr{Op: NOT, Operand: CallExpr{...}}`
- `a || b && c` → `BinaryExpr{Op: OR, Left: a, Right: BinaryExpr{Op: AND, ...}}`

---

### フェーズ 3: AST評価器の実装
**目標**: ASTを走査して真偽値を返す評価器を実装

**ファイル**: `internal/expression/evaluator.go` (既存ファイルを拡張)

**タスク**:
1. [ ] `EvaluationContext` 構造体を定義
   ```go
   type EvaluationContext struct {
       Env          map[string]string
       StepsContext map[string]*StepContext
       JobSuccess   bool
       WorkDir      string  // hashFiles() 用
   }
   ```
2. [ ] `evaluateNode(node Node, ctx *EvaluationContext) (interface{}, error)` を実装
3. [ ] BinaryExpr の評価（`&&`, `||`, `==`, `!=`）
4. [ ] UnaryExpr の評価（`!`）
5. [ ] 既存のステータス関数を CallExpr として評価
6. [ ] 識別子の解決（`env.VAR`, `steps.id.outcome`）
7. [ ] 既存の `Evaluate()` メソッドを新しい実装に切り替え
8. [ ] 後方互換性のテスト（既存テストが全て通ること）

---

### フェーズ 4: 論理演算子 (`&&`, `||`, `!`) の統合テスト
**目標**: 論理演算子が正しく動作することを確認

**ファイル**: `internal/expression/evaluator_test.go` を拡張

**テストケース**:
```go
// AND演算子
{"success() && true", true, true}
{"success() && failure()", true, false}
{"failure() && true", true, false}

// OR演算子
{"success() || failure()", true, true}
{"failure() || failure()", true, false}
{"failure() || true", true, true}

// NOT演算子
{"!failure()", true, true}
{"!success()", true, false}
{"!true", true, false}

// 複合条件
{"success() && !failure()", true, true}
{"(success() || failure()) && true", true, true}
{"success() && env.MY_VAR == 'prod'", {"MY_VAR": "prod"}, true}

// 短絡評価
{"false && undefined_func()", true, false}  // 右辺は評価されない
{"true || undefined_func()", true, true}    // 右辺は評価されない
```

---

### フェーズ 5: `contains()` 関数の実装
**目標**: 文字列や配列に値が含まれるかチェック

**構文**:
```yaml
if: contains(github.event.comment.body, '/deploy')
if: contains(github.actor, 'bot')
```

**実装詳細**:
```go
func evalContains(args []interface{}) (bool, error) {
    if len(args) != 2 {
        return false, fmt.Errorf("contains() requires 2 arguments")
    }
    haystack := toString(args[0])
    needle := toString(args[1])
    return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle)), nil
}
```

**タスク**:
1. [ ] `contains()` を CallExpr 評価に追加
2. [ ] 大文字/小文字を無視した比較を実装
3. [ ] ユニットテスト作成

**テストケース**:
```go
{"contains('Hello World', 'world')", true}   // case-insensitive
{"contains('Hello', 'xyz')", false}
{"contains(env.MESSAGE, 'deploy')", env: {"MESSAGE": "please deploy"}, true}
```

---

### フェーズ 6: `startsWith()` 関数の実装
**目標**: 文字列が特定のプレフィックスで始まるかチェック

**構文**:
```yaml
if: startsWith(github.ref, 'refs/tags/')
if: startsWith(github.event.head_commit.message, 'feat:')
```

**実装詳細**:
```go
func evalStartsWith(args []interface{}) (bool, error) {
    if len(args) != 2 {
        return false, fmt.Errorf("startsWith() requires 2 arguments")
    }
    str := toString(args[0])
    prefix := toString(args[1])
    return strings.HasPrefix(strings.ToLower(str), strings.ToLower(prefix)), nil
}
```

**タスク**:
1. [ ] `startsWith()` を CallExpr 評価に追加
2. [ ] 大文字/小文字を無視した比較を実装
3. [ ] ユニットテスト作成

**テストケース**:
```go
{"startsWith('refs/tags/v1', 'refs/tags/')", true}
{"startsWith('Hello', 'he')", true}   // case-insensitive
{"startsWith('Hello', 'World')", false}
```

---

### フェーズ 7: `endsWith()` 関数の実装（オプション）
**目標**: 文字列が特定のサフィックスで終わるかチェック

**タスク**:
1. [ ] `endsWith()` を CallExpr 評価に追加
2. [ ] ユニットテスト作成

---

### フェーズ 8: `hashFiles()` 関数の実装
**目標**: ファイルのハッシュ値を計算してキャッシュキーなどに使用

**構文**:
```yaml
if: hashFiles('**/package-lock.json') != ''
```

**実装詳細**:
```go
func evalHashFiles(args []interface{}, workDir string) (string, error) {
    if len(args) == 0 {
        return "", nil
    }

    var allBytes []byte
    for _, arg := range args {
        pattern := toString(arg)
        matches, err := filepath.Glob(filepath.Join(workDir, pattern))
        if err != nil {
            continue
        }
        sort.Strings(matches)  // 順序を安定させる
        for _, match := range matches {
            data, err := os.ReadFile(match)
            if err != nil {
                continue
            }
            allBytes = append(allBytes, data...)
        }
    }

    if len(allBytes) == 0 {
        return "", nil  // ファイルがない場合は空文字
    }

    hash := sha256.Sum256(allBytes)
    return hex.EncodeToString(hash[:]), nil
}
```

**タスク**:
1. [ ] `hashFiles()` を CallExpr 評価に追加
2. [ ] glob パターンマッチングを実装
3. [ ] SHA256 ハッシュ計算を実装
4. [ ] ファイルが見つからない場合は空文字を返す
5. [ ] ユニットテスト作成（テスト用の一時ファイルを使用）

**テストケース**:
```go
// テスト用ファイルを作成して検証
{"hashFiles('test.txt') != ''", true}   // ファイルあり
{"hashFiles('nonexistent.txt') != ''", false}  // ファイルなし
```

---

### フェーズ 9: 既存 API との統合と最終テスト
**目標**: 新実装を既存のワークフロー実行に統合

**タスク**:
1. [ ] `ConditionEvaluator.Evaluate()` を新パーサー/評価器に切り替え
2. [ ] 古い正規表現ロジックを削除またはフォールバックとして保持
3. [ ] `step_executor.go` との統合テスト
4. [ ] E2Eテストの追加/更新
5. [ ] エラーメッセージの改善（パースエラー時に位置を表示）

---

### フェーズ 10: ドキュメントとクリーンアップ
**目標**: コードの品質とドキュメントを整備

**タスク**:
1. [ ] GoDoc コメントを全ての公開APIに追加
2. [ ] README に対応構文一覧を追加
3. [ ] 未使用コードの削除
4. [ ] `go vet` / `staticcheck` / `golangci-lint` を通す

---

## ファイル構成（最終形）

```
internal/expression/
├── tokenizer.go       # フェーズ1: トークナイザー
├── tokenizer_test.go
├── ast.go             # フェーズ2: AST定義
├── parser.go          # フェーズ2: パーサー
├── parser_test.go
├── evaluator.go       # フェーズ3: 評価器（拡張）
├── evaluator_test.go  # 拡張
└── functions.go       # フェーズ5-8: 組み込み関数
```

---

## 実装順序の理由

1. **トークナイザー → パーサー → 評価器** の順で基盤を構築
2. **論理演算子** を先に実装（複合条件の基礎）
3. **文字列関数** (`contains`, `startsWith`) を実装（比較的シンプル）
4. **`hashFiles()`** を最後に（ファイルシステムアクセスが必要で複雑）

---

## リスクと対策

| リスク | 対策 |
|--------|------|
| パーサーのバグ | 各フェーズで徹底的にテスト |
| 後方互換性の破壊 | 既存テストを全て維持、フォールバック用意 |
| パフォーマンス低下 | ベンチマークテストを追加 |
| GitHub Actions仕様との差異 | 公式ドキュメントを参照しながら実装 |

---

## 参考リンク
- [GitHub Actions 式の構文](https://docs.github.com/en/actions/learn-github-actions/expressions)
- [ステータス関数](https://docs.github.com/en/actions/learn-github-actions/expressions#status-check-functions)
