# MySQL Schema MCP Server

MySQLデータベースのスキーマ情報を圧縮して提供するModel Context Protocol (MCP) サーバーです。
規模の大きいデータベーススキーマを使っているためにスキーマダンプファイルがコンテキストサイズに載らない場合に、このMCPサーバーが有用です。

https://github.com/user-attachments/assets/f81b2513-31bd-4a60-9b54-45f76323d112

## 提供するツール

- テーブル一覧の取得 (`list_tables`)
  - 指定したデータベース内のすべてのテーブル情報を一覧表示します。テーブル名、コメント、主キー、一意キー、外部キー情報などが含まれます。
  - パラメータ
    - `dbName`: 情報を取得するデータベース名
- テーブル詳細の取得 (`describe_tables`)
  - 指定したデータベースの特定テーブルの詳細情報を表示します。カラム定義、キー制約、インデックスなどの情報を整形して提供します。
  - パラメータ
    - `dbName`: 情報を取得するデータベース名
    - `tableNames`: 詳細情報を取得するテーブル名の配列

## クイックスタート
1. コマンドをインストール

    ```
    go install github.com/shibayu36/mysql-schema-explorer-mcp@latest
    ```

2. mcp.jsonを設定

    ```json
    {
      "mcpServers": {
        "mysql-schema-explorer-mcp": {
          "command": "/path/to/mysql-schema-explorer-mcp",
          "env": {
            "DB_HOST": "127.0.0.1",
            "DB_PORT": "3306",
            "DB_USER": "root",
            "DB_PASSWORD": "root"
          },
        }
      }
    }
    ```

3. エージェントを利用してSQL生成を実行

    例: ecshopデータベースの構造を使って、ユーザー名がshibayu36が最近注文した商品名3つを出して
