# DB_NAME環境変数による固定データベースモード実装

## 概要
mysql-schema-explorer-mcpに、特定のデータベースのみを操作対象とする「固定データベースモード」を実装する。DB_NAME環境変数を設定することで、そのデータベースのみにアクセスを制限し、セキュリティと利便性を向上させる。

## 背景
現在の実装では、ツール呼び出し時に毎回`dbName`パラメータを指定する必要がある。単一のデータベースのみを扱う環境では、これは冗長であり、誤って他のデータベースにアクセスするリスクもある。

## 使い勝手の設計

### 動作仕様

#### 1. DB_NAME環境変数が設定されている場合（固定モード）
- 指定されたデータベースのみを操作対象とする
- ツール呼び出し時の`dbName`パラメータは無視される
- セキュリティ向上：誤って他のデータベースにアクセスすることを防ぐ

#### 2. DB_NAME環境変数が設定されていない場合（通常モード）
- 従来通り、ツール呼び出し時に`dbName`パラメータが必須
- 複数のデータベースを柔軟に操作可能

### 利用例

```bash
# 固定モード
export DB_NAME=myapp_production
export DB_USER=myuser
export DB_PASSWORD=mypassword
mysql-schema-explorer-mcp

# 通常モード（DB_NAMEを設定しない）
export DB_USER=myuser
export DB_PASSWORD=mypassword
mysql-schema-explorer-mcp
```

## コード設計

### 1. アーキテクチャ方針
Handler構造体に固定DB名を持たせる設計を採用する。これにより、Handlerが固定モードかどうかを判断し、適切なDB名を使用できる。

```go
type Handler struct {
    db *DB
    fixedDBName string // 追加：空文字の場合は通常モード
}
```

### 2. 実装の流れ

#### main.go
- `DB_NAME`環境変数を読み込む
- `NewHandler(db, fixedDBName)`でHandlerを初期化
- ツール定義は変更せず、後方互換性を維持

#### handler.go
- コンストラクタを修正：`NewHandler(db *DB, fixedDBName string) *Handler`
- `fixedDBName`フィールドを追加
- DB名決定ロジック：
  ```go
  // ListTablesメソッド内
  dbName := h.fixedDBName
  if dbName == "" {
      // 通常モード：リクエストパラメータから取得
      dbNameRaw, ok := request.Params.Arguments["dbName"]
      if !ok {
          return mcp.NewToolResultError("Database name is not specified"), nil
      }
      dbName, ok = dbNameRaw.(string)
      if !ok || dbName == "" {
          return mcp.NewToolResultError("Database name is not specified correctly"), nil
      }
  }
  ```

### 3. エラーハンドリング
- 固定モードでは、DB_NAMEで指定されたデータベースが存在しない場合、適切なエラーメッセージを返す
- 通常モードでは、従来通りのエラーハンドリングを行う

### 4. テスト戦略

#### 単体テスト（handler_test.go）
- 固定モード時のテストケース
  - DB_NAMEが設定されている場合、リクエストパラメータを無視することを確認
  - 正しいデータベースにアクセスすることを確認
- 通常モード時のテストケース
  - DB_NAMEが設定されていない場合、リクエストパラメータが必須であることを確認
  - 正しいエラーハンドリングを確認

#### E2Eテスト（e2e_test.go）
- 実際のMCPプロトコルで固定モードが動作することを確認
- 環境変数の設定/未設定で動作が切り替わることを確認

## 実装タスク
1. main.goでDB_NAME環境変数を読み込み、Handlerに渡す実装
2. handler.goにfixedDBNameフィールドを追加し、固定モード対応
3. handler_test.goに固定モードのテストケースを追加
4. e2e_test.goに固定モードのE2Eテストを追加
5. README.mdとCLAUDE.mdに固定モードの説明を追加

## 今後の拡張可能性
- 複数のデータベースをカンマ区切りで指定できるようにする
- データベース名のワイルドカード対応（例：`test_*`）
- 読み取り専用モードとの組み合わせ