# MySQL Schema MCP Server

MySQLデータベースのスキーマ情報を提供するModel Context Protocol (MCP) サーバーです。

## 概要

このツールはClaudeなどのLLMクライアントから接続され、データベースのスキーマ情報を簡単に取得できるようにするためのものです。

LLMはこのツールを通じて以下のことができるようになります：

- データベース内のテーブル一覧を取得
- 特定のテーブルの詳細情報（カラム、キー制約、インデックスなど）を確認

## 使用方法

1. 環境変数の設定：
   ```
   DB_HOST=データベースのホスト名
   DB_PORT=データベースのポート番号
   DB_USER=データベースのユーザー名
   DB_PASSWORD=データベースのパスワード
   ```

2. サーバーの起動：
   ```
   go run main.go
   ```

## 提供するツール

### 1. テーブル一覧の取得 (`list_tables`)

指定したデータベース内のすべてのテーブル情報を一覧表示します。テーブル名、コメント、主キー、一意キー、外部キー情報などが含まれます。

**パラメータ**:
- `dbName`: 情報を取得するデータベース名

### 2. テーブル詳細の取得 (`describe_tables`)

指定したデータベースの特定テーブルの詳細情報を表示します。カラム定義、キー制約、インデックスなどの情報を整形して提供します。

**パラメータ**:
- `dbName`: 情報を取得するデータベース名
- `tableNames`: 詳細情報を取得するテーブル名の配列

## 技術情報

- 実装言語: Go
- 使用ライブラリ: github.com/mark3labs/mcp-go
- 通信方式: 標準入出力（stdio）

## Claude Desktopでの利用方法

Claude Desktopでこのツールを利用するには、以下の設定を行います：

1. Claude Desktopの設定ファイル (`~/Library/Application Support/Claude/claude_desktop_config.json`) を編集
2. 以下のように設定を追加

```json
{
  "mcpServers": {
    "mysql-schema": {
      "command": "サーバー実行コマンドのパス",
      "args": [],
      "env": {
        "DB_HOST": "データベースのホスト名",
        "DB_PORT": "データベースのポート番号",
        "DB_USER": "データベースのユーザー名",
        "DB_PASSWORD": "データベースのパスワード"
      }
    }
  }
}
``` 

## 使用例

### テーブル一覧の取得

```json
{
  "dbName": "my_database"
}
```

### テーブル詳細の取得

```json
{
  "dbName": "my_database",
  "tableNames": ["users", "products"]
}
``` 
