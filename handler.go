package main

import (
	"context"
	"fmt"
	"strings"

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

	// フォーマット済みのテキスト出力を構築
	var sb strings.Builder

	// ヘッダー部分
	sb.WriteString(fmt.Sprintf("データベース「%s」のテーブル一覧 (全%d件)\n", dbName, len(tables)))
	sb.WriteString("フォーマット: テーブル名 - テーブルコメント [PK: 主キー] [UK: 一意キー1; 一意キー2...] [FK: 外部キー -> 参照先テーブル.カラム; ...]\n")
	sb.WriteString("※ 複合キー（複数カラムで構成されるキー）は括弧でグループ化: (col1, col2)\n")
	sb.WriteString("※ 複数の異なるキー制約はセミコロンで区切り: key1; key2\n\n")

	// テーブルリスト
	for _, table := range tables {
		sb.WriteString(fmt.Sprintf("- %s - %s", table.Name, table.Comment))

		if len(table.PK) > 0 {
			pkStr := strings.Join(table.PK, ", ")
			if len(table.PK) > 1 {
				pkStr = fmt.Sprintf("(%s)", pkStr)
			}
			sb.WriteString(fmt.Sprintf(" [PK: %s]", pkStr))
		}

		if len(table.UK) > 0 {
			var ukInfo []string
			for _, uk := range table.UK {
				if len(uk.Columns) > 1 {
					ukInfo = append(ukInfo, fmt.Sprintf("(%s)", strings.Join(uk.Columns, ", ")))
				} else {
					ukInfo = append(ukInfo, strings.Join(uk.Columns, ", "))
				}
			}
			sb.WriteString(fmt.Sprintf(" [UK: %s]", strings.Join(ukInfo, "; ")))
		}

		if len(table.FK) > 0 {
			var fkInfo []string
			for _, fk := range table.FK {
				colStr := strings.Join(fk.Columns, ", ")
				refColStr := strings.Join(fk.RefColumns, ", ")

				if len(fk.Columns) > 1 {
					colStr = fmt.Sprintf("(%s)", colStr)
				}

				if len(fk.RefColumns) > 1 {
					refColStr = fmt.Sprintf("(%s)", refColStr)
				}

				fkInfo = append(fkInfo, fmt.Sprintf("%s -> %s.%s",
					colStr,
					fk.RefTable,
					refColStr))
			}
			sb.WriteString(fmt.Sprintf(" [FK: %s]", strings.Join(fkInfo, "; ")))
		}

		sb.WriteString("\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
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

	var sb strings.Builder

	// すべてのテーブルに対して情報を取得
	for i, tableName := range tableNames {
		// 2つ目以降のテーブルの前に区切り線を追加
		if i > 0 {
			sb.WriteString("\n---\n\n")
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
			sb.WriteString(fmt.Sprintf("# テーブル: %s\nテーブルが見つかりません\n", tableName))
			continue
		}

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

		// 結果の整形
		sb.WriteString(fmt.Sprintf("# テーブル: %s", tableName))
		if tableInfo.Comment != "" {
			sb.WriteString(fmt.Sprintf(" - %s", tableInfo.Comment))
		}
		sb.WriteString("\n\n")

		sb.WriteString("## カラム\n")
		for _, col := range columns {
			nullable := "NOT NULL"
			if col.IsNullable == "YES" {
				nullable = "NULL"
			}

			defaultValue := ""
			if col.Default.Valid {
				defaultValue = fmt.Sprintf(" DEFAULT %s", col.Default.String)
			}

			comment := ""
			if col.Comment != "" {
				comment = fmt.Sprintf(" [%s]", col.Comment)
			}

			sb.WriteString(fmt.Sprintf("- %s: %s %s%s%s\n",
				col.Name, col.Type, nullable, defaultValue, comment))
		}
		sb.WriteString("\n")

		sb.WriteString("## キー情報\n")

		if len(primaryKeys) > 0 {
			pkStr := strings.Join(primaryKeys, ", ")
			if len(primaryKeys) > 1 {
				pkStr = fmt.Sprintf("(%s)", pkStr)
			}
			sb.WriteString(fmt.Sprintf("[PK: %s]\n", pkStr))
		}

		if len(uniqueKeys) > 0 {
			var ukInfo []string
			for _, uk := range uniqueKeys {
				if len(uk.Columns) > 1 {
					ukInfo = append(ukInfo, fmt.Sprintf("(%s)", strings.Join(uk.Columns, ", ")))
				} else {
					ukInfo = append(ukInfo, strings.Join(uk.Columns, ", "))
				}
			}
			sb.WriteString(fmt.Sprintf("[UK: %s]\n", strings.Join(ukInfo, "; ")))
		}

		if len(foreignKeys) > 0 {
			var fkInfo []string
			for _, fk := range foreignKeys {
				colStr := strings.Join(fk.Columns, ", ")
				refColStr := strings.Join(fk.RefColumns, ", ")

				if len(fk.Columns) > 1 {
					colStr = fmt.Sprintf("(%s)", colStr)
				}

				if len(fk.RefColumns) > 1 {
					refColStr = fmt.Sprintf("(%s)", refColStr)
				}

				fkInfo = append(fkInfo, fmt.Sprintf("%s -> %s.%s",
					colStr,
					fk.RefTable,
					refColStr))
			}
			sb.WriteString(fmt.Sprintf("[FK: %s]\n", strings.Join(fkInfo, "; ")))
		}

		if len(indexes) > 0 {
			var idxInfo []string
			for _, idx := range indexes {
				if len(idx.Columns) > 1 {
					idxInfo = append(idxInfo, fmt.Sprintf("(%s)", strings.Join(idx.Columns, ", ")))
				} else {
					idxInfo = append(idxInfo, strings.Join(idx.Columns, ", "))
				}
			}
			sb.WriteString(fmt.Sprintf("[INDEX: %s]\n", strings.Join(idxInfo, "; ")))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}
