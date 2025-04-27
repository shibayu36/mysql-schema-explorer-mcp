package main

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"
)

// Handler MCPハンドラーを実装する構造体
type Handler struct {
	db *DB
}

func NewHandler(db *DB) *Handler {
	return &Handler{db: db}
}

// ListTables 全てのテーブルのサマリー情報を返す
func (h *Handler) ListTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbNameRaw, ok := request.Params.Arguments["dbName"]
	if !ok {
		return mcp.NewToolResultError("データベース名が指定されていません"), nil
	}

	dbName, ok := dbNameRaw.(string)
	if !ok || dbName == "" {
		return mcp.NewToolResultError("データベース名が正しく指定されていません"), nil
	}

	// テーブル情報の取得
	tables, err := h.db.FetchAllTableSummaries(ctx, dbName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("テーブル情報の取得に失敗しました: %v", err)), nil
	}

	// テーブルが見つからない
	if len(tables) == 0 {
		return mcp.NewToolResultText("データベース内にテーブルが存在しません。"), nil
	}

	// 出力の作成
	var output bytes.Buffer
	{
		tmpl, err := template.New("listTables").Funcs(funcMap).Parse(listTablesTemplate)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("テンプレートの解析に失敗しました: %v", err)), nil
		}

		if err := tmpl.Execute(&output, ListTablesData{
			DBName: dbName,
			Tables: tables,
		}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("テンプレートの実行に失敗しました: %v", err)), nil
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}

// DescribeTables は指定されたテーブルの詳細情報を返すハンドラーメソッド
func (h *Handler) DescribeTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// dbNameパラメータを取得
	dbNameRaw, ok := request.Params.Arguments["dbName"]
	if !ok {
		return mcp.NewToolResultError("データベース名が指定されていません"), nil
	}

	dbName, ok := dbNameRaw.(string)
	if !ok || dbName == "" {
		return mcp.NewToolResultError("データベース名が正しく指定されていません"), nil
	}

	// テーブル名一覧を作成
	tableNamesRaw, ok := request.Params.Arguments["tableNames"]
	if !ok {
		return mcp.NewToolResultError("テーブル名が指定されていません"), nil
	}
	tableNamesInterface, ok := tableNamesRaw.([]interface{})
	if !ok || len(tableNamesInterface) == 0 {
		return mcp.NewToolResultError("テーブル名の配列が正しく指定されていません"), nil
	}
	var tableNames []string
	for _, v := range tableNamesInterface {
		if tableName, ok := v.(string); ok && tableName != "" {
			tableNames = append(tableNames, tableName)
		}
	}
	if len(tableNames) == 0 {
		return mcp.NewToolResultError("有効なテーブル名が指定されていません"), nil
	}

	allTables, err := h.db.FetchAllTableSummaries(ctx, dbName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("テーブル情報の取得に失敗しました: %v", err)), nil
	}

	// 出力の準備
	var output bytes.Buffer
	tmpl, err := template.New("describeTableDetail").Funcs(funcMap).Parse(describeTableDetailTemplate)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("テンプレートの解析に失敗しました: %v", err)), nil
	}

	// すべてのテーブルに対して情報を取得
	for i, tableName := range tableNames {
		// 2つ目以降のテーブルの前に区切り線を追加
		if i > 0 {
			output.WriteString("\n---\n\n")
		}

		// 指定されたテーブルを探す
		var tableInfo TableSummary
		var tableFound bool
		for _, t := range allTables {
			if t.Name == tableName {
				tableInfo = t
				tableFound = true
				break
			}
		}

		if !tableFound {
			output.WriteString(fmt.Sprintf("# テーブル: %s\nテーブルが見つかりません\n", tableName))
			continue
		}

		// テーブル詳細情報の取得
		primaryKeys, err := h.db.FetchPrimaryKeys(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("主キー情報の取得に失敗しました: %v", err)), nil
		}

		uniqueKeys, err := h.db.FetchUniqueKeys(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("一意キー情報の取得に失敗しました: %v", err)), nil
		}

		foreignKeys, err := h.db.FetchForeignKeys(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("外部キー情報の取得に失敗しました: %v", err)), nil
		}

		columns, err := h.db.FetchTableColumns(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("カラム情報の取得に失敗しました: %v", err)), nil
		}

		indexes, err := h.db.FetchTableIndexes(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("インデックス情報の取得に失敗しました: %v", err)), nil
		}

		// テンプレートに渡すデータを作成
		tableDetail := TableDetail{
			Name:        tableName,
			Comment:     tableInfo.Comment,
			Columns:     columns,
			PrimaryKeys: primaryKeys,
			UniqueKeys:  uniqueKeys,
			ForeignKeys: foreignKeys,
			Indexes:     indexes,
		}

		// テンプレートを実行してバッファに書き込む
		if err := tmpl.Execute(&output, tableDetail); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("テンプレートの実行に失敗しました: %v", err)), nil
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}
