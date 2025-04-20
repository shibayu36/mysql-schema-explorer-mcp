# MySQL Schema MCP Server 要件定義

## 概要
このプロジェクトでは、Model Context Protocol（MCP）を使用してMySQLデータベースのスキーマ情報を提供するサーバーを実装します。このサーバーはClaudeなどのLLMクライアントから接続され、データベーススキーマに関する情報を取得するツールを提供します。

## 環境変数設定
- **DB_HOST**: データベースのホスト名
- **DB_PORT**: データベースのポート番号
- **DB_USER**: データベースのユーザー名
- **DB_PASSWORD**: データベースのパスワード
- **DB_NAME**: 接続するデータベース名

## MCPサーバーのツール

1. **テーブル一覧を取得**
- `list_tables`
- 説明: 指定されたデータベースの全テーブル名をリストとして返す
- 引数: なし
- 戻り値: テーブル名とテーブルコメント、キー情報のリスト（テキスト形式）
- 出力フォーマット:
  ```
  データベース「DB_NAME」のテーブル一覧 (全X件)
  フォーマット: テーブル名 - テーブルコメント [PK: 主キー] [UK: 一意キー1; 一意キー2...] [FK: 外部キー -> 参照先テーブル.カラム; ...]
  ※ 複合キー（複数カラムで構成されるキー）は括弧でグループ化: (col1, col2)
  ※ 複数の異なるキー制約はセミコロンで区切り: key1; key2
  
  - users - ユーザー情報 [PK: id] [UK: email; username] [FK: role_id -> roles.id; department_id -> departments.id]
  - posts - 投稿情報 [PK: id] [UK: slug] [FK: user_id -> users.id; category_id -> categories.id]
  - order_items - 注文商品 [PK: (order_id, item_id)] [FK: (order_id, item_id) -> orders.(id, item_id); product_id -> products.id]
  ```

2. **テーブル詳細を取得**
- `describe_tables`
- 説明: 指定されたテーブルのカラム情報、インデックス、外部キー制約などの詳細情報を返す
- 引数: tableNames（string配列）- 詳細情報を取得するテーブル名（複数指定可能）
- 戻り値: 各テーブルの詳細情報を整形したテキスト
- 出力フォーマット:
  ```
  # テーブル: order_items - 注文商品

  ## カラム
  - order_id: int(11) NOT NULL [注文ID]
  - item_id: int(11) NOT NULL [商品ID]
  - product_id: int(11) NOT NULL [製品ID]
  - quantity: int(11) NOT NULL [数量]
  - price: decimal(10,2) NOT NULL [価格]
  - user_id: int(11) NOT NULL [ユーザーID]

  ## キー情報
  [PK: (order_id, item_id)]
  [UK: (user_id, product_id)]
  [FK: (order_id, item_id) -> orders.(id, item_id); product_id -> products.id; user_id -> users.id]
  [INDEX: price; quantity]

  ---

  # テーブル: users - ユーザー情報

  ## カラム
  - id: int(11) NOT NULL [ユーザーID]
  - username: varchar(50) NOT NULL [ユーザー名]
  - email: varchar(100) NOT NULL [メールアドレス]
  - password: varchar(255) NOT NULL [パスワード]
  - created_at: timestamp NULL DEFAULT CURRENT_TIMESTAMP [作成日時]

  ## キー情報
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
- [ ] サーバーの動作テスト
