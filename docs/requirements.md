# MySQL Schema MCP Server 要件定義

## 概要
このプロジェクトでは、Model Context Protocol（MCP）を使用してMySQLデータベースのスキーマ情報を提供するサーバーを実装します。このサーバーはClaudeなどのLLMクライアントから接続され、データベーススキーマに関する情報を取得するツールを提供します。

## 環境変数設定
- **DB_HOST**: データベースのホスト名
- **DB_PORT**: データベースのポート番号
- **DB_USER**: データベースのユーザー名
- **DB_PASSWORD**: データベースのパスワード

## MCPサーバーのツール

1. **テーブル一覧を取得**
- `list_tables`
- 説明: 指定されたデータベースの全テーブル名をリストとして返す
- 引数: dbName (string) - 情報を取得するデータベース名
- 戻り値: テーブル名とテーブルコメント、キー情報のリスト（テキスト形式）
- 出力フォーマット:
  ```text
  Tables in database "DB_NAME" (Total: X)
  Format: Table Name - Table Comment [PK: Primary Key] [UK: Unique Key 1; Unique Key 2...] [FK: Foreign Key -> Referenced Table.Column; ...]
  * Composite keys (keys composed of multiple columns) are grouped in parentheses: (col1, col2)
  * Multiple different key constraints are separated by semicolons: key1; key2

  - users - User information [PK: id] [UK: email; username] [FK: role_id -> roles.id; department_id -> departments.id]
  - posts - Post information [PK: id] [UK: slug] [FK: user_id -> users.id; category_id -> categories.id]
  - order_items - Order items [PK: (order_id, item_id)] [FK: (order_id, item_id) -> orders.(id, item_id); product_id -> products.id]
  ```

2. **テーブル詳細を取得**
- `describe_tables`
- 説明: 指定されたテーブルのカラム情報、インデックス、外部キー制約などの詳細情報を返す
- 引数: 
  - dbName (string) - 情報を取得するデータベース名
  - tableNames（string配列）- 詳細情報を取得するテーブル名（複数指定可能）
- 戻り値: 各テーブルの詳細情報を整形したテキスト
- 出力フォーマット:
  ```text
  # Table: order_items - Order Items

  ## Columns
  - order_id: int(11) NOT NULL [Order ID]
  - item_id: int(11) NOT NULL [Item ID]
  - product_id: int(11) NOT NULL [Product ID]
  - quantity: int(11) NOT NULL [Quantity]
  - price: decimal(10,2) NOT NULL [Price]
  - user_id: int(11) NOT NULL [User ID]

  ## Key Information
  [PK: (order_id, item_id)]
  [UK: (user_id, product_id)]
  [FK: (order_id, item_id) -> orders.(id, item_id); product_id -> products.id; user_id -> users.id]
  [INDEX: price; quantity]

  ---

  # Table: users - User Information

  ## Columns
  - id: int(11) NOT NULL [User ID]
  - username: varchar(50) NOT NULL [Username]
  - email: varchar(100) NOT NULL [Email Address]
  - password: varchar(255) NOT NULL [Password]
  - created_at: timestamp NULL DEFAULT CURRENT_TIMESTAMP [Created At]

  ## Key Information
  [PK: id]
  [UK: email; username]
  [INDEX: created_at]
  ```

  複数のテーブルを指定した場合、各テーブル情報の間に区切り線（`---`）が挿入されます。

## 実装の流れ

1. **プロジェクトセットアップ**
- MCPライブラリのインストール
- 必要な依存関係のインストール（MySQLクライアントライブラリなど）

2. **MCPサーバーの初期化**
- サーバーインスタンスの作成と名前の設定

3. **環境変数の読み込み**
- サーバー起動時に環境変数を読み込み、データベース接続情報を設定

4. **データベース接続ヘルパー**
- データベース接続を管理するヘルパー機能の実装
- データベース名はツール呼び出し時の引数として受け取る

5. **ツールの実装**
- 各ツール機能の実装
- ツール内で適切なデータベースクエリを実行し、結果を整形

6. **サーバーの実行**
- 標準入出力（stdio）を使用してクライアントと通信するようにサーバーを設定

## 進捗状況

- [x] プロジェクトセットアップ
- [x] MCPサーバーの初期化
- [x] 環境変数の読み込み
- [x] データベース接続ヘルパーの実装
- [x] `list_tables`ツールの実装
- [x] `describe_tables`ツールの実装
- [x] DB_NAMEは環境変数ではなく、tool callの引数で受け取るように
- [ ] セキュリティ周りの調整
