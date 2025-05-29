# DB専用MCPサーバー機能の設計

## 概要
環境変数`DB_NAME`を指定することで、特定のデータベース専用のMCPサーバーとして動作させる機能を実装する。

## 背景
現在の実装では、ツールを呼び出す際に毎回`dbName`パラメータを指定する必要がある。特定のデータベースのみを扱うケースでは、環境変数で対象DBを固定することで、より使いやすくなる。

## 設計方針

### 1. 環境変数の追加
- `DB_NAME`: 対象とするデータベース名（オプション）
- 既存の環境変数（`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`）と同様に扱う

### 2. 動作モード
#### DB_NAME指定時（専用モード）
- MCPサーバーは指定されたデータベース専用として動作
- ツールの`dbName`パラメータは不要になる（指定されても無視）
- サーバー起動時に指定されたデータベースの存在を確認
- ツールの説明文に対象DB名を含める（例：「データベース`myapp`のテーブル情報を返します」）

#### DB_NAME未指定時（汎用モード）
- 現在の動作を維持
- ツールの`dbName`パラメータが必須
- 複数のデータベースを扱える

### 3. 実装詳細

#### DBConfig構造体の拡張
```go
type DBConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string // 新規追加: 対象DB名（オプション）
}
```

#### Handler構造体の拡張
```go
type Handler struct {
    db       *DB
    dbName   string // 新規追加: 固定DB名（専用モード時に設定）
}
```

#### ツール定義の動的生成
```go
// main.go内でDB_NAME指定時は説明文を変更
if dbName != "" {
    listTables := mcp.NewTool(
        "list_tables",
        mcp.WithDescription(fmt.Sprintf("Returns a list of table information in the MySQL database '%s'.", dbName)),
        // dbNameパラメータは定義しない
    )
} else {
    // 現在の定義を維持
}
```

#### ハンドラーメソッドの修正
```go
func (h *Handler) ListTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    var dbName string
    
    // 専用モード時は固定DB名を使用
    if h.dbName != "" {
        dbName = h.dbName
    } else {
        // 汎用モード時は従来通りパラメータから取得
        dbNameRaw, ok := request.Params.Arguments["dbName"]
        if !ok {
            return mcp.NewToolResultError("Database name is not specified"), nil
        }
        // ... 既存の検証処理
    }
    
    // 以降の処理は同じ
}
```

### 4. エラーハンドリング
- DB_NAME指定時、起動時にデータベースの存在を確認
- 存在しない場合は起動エラーとする
- 接続確認用の関数を追加：
```go
func verifyDatabase(conn *sql.DB, dbName string) error {
    var exists int
    query := "SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?"
    err := conn.QueryRow(query, dbName).Scan(&exists)
    if err != nil {
        return fmt.Errorf("failed to verify database: %v", err)
    }
    if exists == 0 {
        return fmt.Errorf("database '%s' does not exist", dbName)
    }
    return nil
}
```

### 5. テストの考慮事項
- DB_NAME指定時の動作確認
  - データベース存在チェック
  - パラメータ無視の確認
  - 説明文の確認
- DB_NAME未指定時の動作確認（現在の動作が維持されること）
- 環境変数のモック化

## 実装手順
1. DBConfig構造体にDBNameフィールドを追加
2. loadDBConfig関数でDB_NAME環境変数を読み込む
3. connectDB関数でデータベース存在確認を追加
4. Handler構造体にdbNameフィールドを追加
5. main.go内でDB_NAME指定時の分岐処理を追加
6. 各ハンドラーメソッドで専用モード時の処理を追加
7. テストケースを追加

## 影響範囲
- 既存の動作には影響なし（DB_NAME未指定時は現在の動作を維持）
- READMEへの使用方法追記が必要
- CLAUDE.mdへの環境変数追記が必要