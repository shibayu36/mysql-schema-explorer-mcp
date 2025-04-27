# ListTables 関数の text/template 化 設計案

## 目的

`handler.go` の `ListTables` 関数の出力を `text/template` を使って生成するようにリファクタリングし、出力フォーマットの変更を容易にする。

## ファイル構成

テンプレート関連の処理（テンプレート文字列、データ構造、ヘルパー関数）を `view.go` に分離する。

## 実装詳細

1.  **テンプレート文字列の定義 (`view.go`)**
    *   `const listTablesTemplate` として定義する。

    ```go
    const listTablesTemplate = `データベース「{{.DBName}}」のテーブル一覧 (全{{len .Tables}}件)
    フォーマット: テーブル名 - テーブルコメント [PK: 主キー] [UK: 一意キー1; 一意キー2...] [FK: 外部キー -> 参照先テーブル.カラム; ...]
    ※ 複合キー（複数カラムで構成されるキー）は括弧でグループ化: (col1, col2)
    ※ 複数の異なるキー制約はセミコロンで区切り: key1; key2

    {{range .Tables -}}
    - {{.Name}} - {{.Comment}}{{formatPK .PK}}{{formatUK .UK}}{{formatFK .FK}}
    {{end -}}
    `
    ```

2.  **テンプレート用データ構造の定義 (`view.go`)**
    *   `ListTablesData` 構造体を定義する。

    ```go
    type ListTablesData struct {
        DBName string
        Tables []TableSummary // db.go で定義されている TableSummary を想定
    }
    ```

3.  **テンプレートヘルパー関数の実装 (`view.go`)**
    *   PK, UK, FK を整形するためのヘルパー関数を `template.FuncMap` として定義する。
        *   `formatPK(pk []string) string`
        *   `formatUK(uk []KeyInfo) string`
        *   `formatFK(fk []ForeignKeyInfo) string`
    *   これらの関数は、現在の `ListTables` 内の `strings.Builder` を使ったロジックを基に実装する。

    ```go
    func formatPK(pk []string) string {
        // 実装...
    }
    func formatUK(uk []KeyInfo) string {
        // 実装...
    }
    func formatFK(fk []ForeignKeyInfo) string {
        // 実装...
    }

    var funcMap = template.FuncMap{
        "formatPK": formatPK,
        "formatUK": formatUK,
        "formatFK": formatFK,
    }
    ```

4.  **`ListTables` 関数の修正 (`handler.go`)**
    *   テーブル情報を取得する。
    *   取得したデータを `ListTablesData` 構造体に格納する。
    *   `text/template` を使ってテンプレートを準備する。
    *   テンプレートを実行し、結果をバッファに書き込む。
    *   結果の文字列を `mcp.NewToolResultText()` で返す。
    *   既存の `strings.Builder` を使ったフォーマット処理は削除する。

## その他

*   `db.go` の `TableSummary`, `KeyInfo`, `ForeignKeyInfo` の定義が `view.go` から参照されるため、アクセス可能にする必要がある。

---

# DescribeTables 関数の text/template 化 設計案

## 目的

`handler.go` の `DescribeTables` 関数の出力を `text/template` を使って生成するようにリファクタリングし、出力フォーマットの変更を容易にする。`ListTables` と同様に、表示ロジックを `view.go` に分離する。

## ファイル構成

*   **`view.go`**: 単一テーブルの詳細を描画するテンプレート、関連するデータ構造 (`TableDetail`)、ヘルパー関数 (`describeFuncMap`) を定義する。
*   **`handler.go`**: `DescribeTables` 関数内でテーブル情報の取得、ループ処理、区切り線や「テーブルが見つかりません」の出力、`view.go` のテンプレート呼び出しを行う。

## 実装詳細

1.  **テンプレート用データ構造の定義 (`view.go`)**
    *   単一テーブルの詳細情報を保持する `TableDetail` 構造体を定義する。`db.go` で定義されている型を利用する。

    ```go
    // TableDetail 個々のテーブルの詳細情報 (db.go の型を使用)
    type TableDetail struct {
        Name        string
        Comment     string
        Columns     []ColumnInfo // db.go の ColumnInfo
        PrimaryKeys []string     // string スライス
        UniqueKeys  []UniqueKey  // db.go の UniqueKey
        ForeignKeys []ForeignKey // db.go の ForeignKey
        Indexes     []IndexInfo  // db.go の IndexInfo
    }
    ```

2.  **テンプレート文字列の定義 (`view.go`)**
    *   単一テーブルの詳細を描画する `describeTableDetailTemplate` を `const` で定義する。

    ```go
    const describeTableDetailTemplate = `# テーブル: {{.Name}}{{if .Comment}} - {{.Comment}}{{end}}

    ## カラム{{range .Columns}}
    {{formatColumn .}}{{end}}

    ## キー情報{{if .PrimaryKeys}}
    [PK: {{formatPK .PrimaryKeys}}]{{end}}{{if .UniqueKeys}}
    [UK: {{formatUK .UniqueKeys}}]{{end}}{{if .ForeignKeys}}
    [FK: {{formatFK .ForeignKeys}}]{{end}}{{if .Indexes}}
    [INDEX: {{formatIndex .Indexes}}]{{end}}
    `
    ```

3.  **テンプレートヘルパー関数の実装 (`view.go`)**
    *   `DescribeTables` 専用のヘルパー関数を含む `describeFuncMap` を定義する。`formatPK`, `formatUK`, `formatFK` は `ListTables` と共通のものを利用する。
        *   `formatColumn(col ColumnInfo) string`: カラム一行分の情報を整形して返す。
        *   `formatIndex(idx []IndexInfo) string`
    *   これらの関数は、現在の `DescribeTables` 内の `strings.Builder` を使ったロジックや、`db.go` の型に合わせて実装する。

    ```go
    // formatColumn, formatIndex の実装...

    var describeFuncMap = template.FuncMap{
        "formatPK":     formatPK, // 共通
        "formatUK":     formatUK, // 共通
        "formatFK":     formatFK, // 共通
        "formatIndex":  formatIndex,
        "formatColumn": formatColumn, // カラム情報を一行に整形
    }
    ```

4.  **`DescribeTables` 関数の修正 (`handler.go`)**
    *   `bytes.Buffer` を用意する。
    *   `describeTableDetailTemplate` と `describeFuncMap` でテンプレートを準備する。
    *   引数で受け取った `tableNames` をループ処理する。
    *   ループの先頭で、2つ目以降のテーブルであれば区切り線 (`
---

`) をバッファに書き込む。
    *   `db.go` の関数を使ってテーブルの存在確認と詳細情報 (Comment, Columns, PK, UK, FK, Indexes) を取得する。
        *   テーブルが見つからない場合は、バッファに「テーブルが見つかりません」というメッセージを書き込み、`continue` する。
        *   情報の取得中にエラーが発生した場合は、処理を中断し、エラー結果 (`mcp.NewToolResultError`) を返す。
    *   取得した情報を `TableDetail` 構造体に詰める。
    *   準備したテンプレートを実行 (`tmpl.Execute`) し、結果をバッファに追記する。
    *   ループ完了後、バッファの内容を `mcp.NewToolResultText()` で返す。
    *   既存の `strings.Builder` を使ったフォーマット処理は削除する。
