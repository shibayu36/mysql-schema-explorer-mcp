package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DBConfig はデータベース接続設定を保持する構造体
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

// TableInfo はテーブル情報を保持する構造体
type TableInfo struct {
	Name    string
	Comment string
	PK      []string     // 主キーカラム
	UK      []UniqueKey  // 一意キー情報
	FK      []ForeignKey // 外部キー情報
}

// UniqueKey は一意キー情報を保持する構造体
type UniqueKey struct {
	Name    string
	Columns []string
}

// ForeignKey は外部キー情報を保持する構造体
type ForeignKey struct {
	Name       string
	Columns    []string
	RefTable   string
	RefColumns []string
}

// ColumnInfo はカラム情報を保持する構造体
type ColumnInfo struct {
	Name       string
	Type       string
	IsNullable string
	Default    sql.NullString
	Comment    string
}

// IndexInfo はインデックス情報を保持する構造体
type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
}

var db *sql.DB

func main() {
	dbConfig, err := loadDBConfig()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	db, err = connectDB(dbConfig)
	if err != nil {
		log.Fatalf("データベース接続に失敗しました: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("データベース接続確認に失敗しました: %v", err)
	}

	s := server.NewMCPServer(
		"mysql-schema-mcp",
		"1.0.0",
	)

	listTables := mcp.NewTool(
		"list_tables",
		mcp.WithDescription("MySQLのデータベース内のテーブル情報を一覧で返す"),
		mcp.WithString("dbName",
			mcp.Required(),
			mcp.Description("情報を取得するデータベース名"),
		),
	)

	s.AddTool(listTables, listTablesHandler)

	describeTables := mcp.NewTool(
		"describe_tables",
		mcp.WithDescription("指定されたテーブルの詳細情報を返す"),
		mcp.WithString("dbName",
			mcp.Required(),
			mcp.Description("情報を取得するデータベース名"),
		),
		mcp.WithArray(
			"tableNames",
			mcp.Items(
				map[string]interface{}{
					"type": "string",
				},
			),
			mcp.Required(),
			mcp.Description("詳細情報を取得するテーブル名(複数指定可能)"),
		),
	)

	s.AddTool(describeTables, describeTablesHandler)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func loadDBConfig() (DBConfig, error) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "3306"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		return DBConfig{}, fmt.Errorf("DB_USER環境変数が設定されていません")
	}

	password := os.Getenv("DB_PASSWORD")

	return DBConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}, nil
}

func connectDB(config DBConfig) (*sql.DB, error) {
	// データベース名を指定せずに接続（各ツール実行時にデータベースを指定する）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/",
		config.User, config.Password, config.Host, config.Port)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func listTablesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// dbNameパラメータを取得
	dbNameRaw, ok := request.Params.Arguments["dbName"]
	if !ok {
		return mcp.NewToolResultError("データベース名が指定されていません"), nil
	}

	dbName, ok := dbNameRaw.(string)
	if !ok || dbName == "" {
		return mcp.NewToolResultError("データベース名が正しく指定されていません"), nil
	}

	// テーブル情報の取得
	tables, err := fetchTablesWithAllInfo(ctx, dbName)
	if err != nil {
		// エラーが発生した場合は適切なエラーメッセージを返す
		return mcp.NewToolResultError(fmt.Sprintf("テーブル情報の取得に失敗しました: %v", err)), nil
	}

	// テーブルが見つからない場合
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
		// 基本情報
		sb.WriteString(fmt.Sprintf("- %s - %s", table.Name, table.Comment))

		// 主キー情報
		if len(table.PK) > 0 {
			// 主キーが複数カラムの場合は括弧でグループ化
			pkStr := strings.Join(table.PK, ", ")
			if len(table.PK) > 1 {
				pkStr = fmt.Sprintf("(%s)", pkStr)
			}
			sb.WriteString(fmt.Sprintf(" [PK: %s]", pkStr))
		}

		// 一意キー情報
		if len(table.UK) > 0 {
			var ukInfo []string
			for _, uk := range table.UK {
				// 複合キーの場合は括弧でグループ化
				if len(uk.Columns) > 1 {
					ukInfo = append(ukInfo, fmt.Sprintf("(%s)", strings.Join(uk.Columns, ", ")))
				} else {
					ukInfo = append(ukInfo, strings.Join(uk.Columns, ", "))
				}
			}
			sb.WriteString(fmt.Sprintf(" [UK: %s]", strings.Join(ukInfo, "; ")))
		}

		// 外部キー情報
		if len(table.FK) > 0 {
			var fkInfo []string
			for _, fk := range table.FK {
				// カラムとリファレンスカラムを整形
				colStr := strings.Join(fk.Columns, ", ")
				refColStr := strings.Join(fk.RefColumns, ", ")

				// 複合キーの場合は括弧でグループ化
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

// fetchTablesWithAllInfo はテーブル名、コメント、および全てのキー情報を取得する関数
func fetchTablesWithAllInfo(ctx context.Context, dbName string) ([]TableInfo, error) {
	// 基本的なテーブル情報を取得
	tables, err := fetchTablesWithComments(ctx, dbName)
	if err != nil {
		return nil, err
	}

	// 各テーブルの追加情報を取得
	for i := range tables {
		// 主キー情報を取得
		tables[i].PK, err = fetchPrimaryKeys(ctx, dbName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		// 一意キー情報を取得
		tables[i].UK, err = fetchUniqueKeys(ctx, dbName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		// 外部キー情報を取得
		tables[i].FK, err = fetchForeignKeys(ctx, dbName, tables[i].Name)
		if err != nil {
			return nil, err
		}
	}

	return tables, nil
}

// fetchTablesWithComments はテーブル名とコメントを取得する関数
func fetchTablesWithComments(ctx context.Context, dbName string) ([]TableInfo, error) {
	query := `
		SELECT 
			TABLE_NAME, 
			IFNULL(TABLE_COMMENT, '') AS TABLE_COMMENT 
		FROM 
			INFORMATION_SCHEMA.TABLES 
		WHERE 
			TABLE_SCHEMA = ? 
		ORDER BY 
			TABLE_NAME
	`

	rows, err := db.QueryContext(ctx, query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		if err := rows.Scan(&table.Name, &table.Comment); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

// fetchPrimaryKeys はテーブルの主キーカラムを取得する関数
func fetchPrimaryKeys(ctx context.Context, dbName string, tableName string) ([]string, error) {
	query := `
		SELECT 
			COLUMN_NAME
		FROM 
			INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
		WHERE 
			CONSTRAINT_SCHEMA = ? 
			AND TABLE_NAME = ? 
			AND CONSTRAINT_NAME = 'PRIMARY'
		ORDER BY 
			ORDINAL_POSITION
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return primaryKeys, nil
}

// fetchUniqueKeys はテーブルの一意キー制約を取得する関数
func fetchUniqueKeys(ctx context.Context, dbName string, tableName string) ([]UniqueKey, error) {
	query := `
		SELECT 
			kcu.CONSTRAINT_NAME,
			kcu.COLUMN_NAME
		FROM 
			INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
		JOIN 
			INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
		ON 
			kcu.CONSTRAINT_SCHEMA = tc.CONSTRAINT_SCHEMA
			AND kcu.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
			AND kcu.TABLE_NAME = tc.TABLE_NAME
		WHERE 
			kcu.TABLE_SCHEMA = ? 
			AND kcu.TABLE_NAME = ? 
			AND tc.CONSTRAINT_TYPE = 'UNIQUE'
		ORDER BY 
			kcu.CONSTRAINT_NAME,
			kcu.ORDINAL_POSITION
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ukMap := make(map[string][]string)
	for rows.Next() {
		var constraintName, columnName string
		if err := rows.Scan(&constraintName, &columnName); err != nil {
			return nil, err
		}
		ukMap[constraintName] = append(ukMap[constraintName], columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var uniqueKeys []UniqueKey
	for name, columns := range ukMap {
		uniqueKeys = append(uniqueKeys, UniqueKey{
			Name:    name,
			Columns: columns,
		})
	}

	return uniqueKeys, nil
}

// fetchForeignKeys はテーブルの外部キー制約を取得する関数
func fetchForeignKeys(ctx context.Context, dbName string, tableName string) ([]ForeignKey, error) {
	query := `
		SELECT 
			kcu.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME
		FROM 
			INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
		JOIN 
			INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS rc
		ON 
			kcu.CONSTRAINT_SCHEMA = rc.CONSTRAINT_SCHEMA
			AND kcu.CONSTRAINT_NAME = rc.CONSTRAINT_NAME
		WHERE 
			kcu.TABLE_SCHEMA = ? 
			AND kcu.TABLE_NAME = ? 
			AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY 
			kcu.CONSTRAINT_NAME,
			kcu.ORDINAL_POSITION
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[string]ForeignKey)
	for rows.Next() {
		var constraintName, columnName, refTableName, refColumnName string
		if err := rows.Scan(&constraintName, &columnName, &refTableName, &refColumnName); err != nil {
			return nil, err
		}

		fk, exists := fkMap[constraintName]
		if !exists {
			fk = ForeignKey{
				Name:     constraintName,
				RefTable: refTableName,
			}
		}

		fk.Columns = append(fk.Columns, columnName)
		fk.RefColumns = append(fk.RefColumns, refColumnName)
		fkMap[constraintName] = fk
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var foreignKeys []ForeignKey
	for _, fk := range fkMap {
		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

// describeTablesHandler は指定されたテーブルの詳細情報を返すハンドラー
func describeTablesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// dbNameパラメータを取得
	dbNameRaw, ok := request.Params.Arguments["dbName"]
	if !ok {
		return mcp.NewToolResultError("データベース名が指定されていません"), nil
	}

	dbName, ok := dbNameRaw.(string)
	if !ok || dbName == "" {
		return mcp.NewToolResultError("データベース名が正しく指定されていません"), nil
	}

	// リクエストからテーブル名の配列を取得
	tableNamesRaw, ok := request.Params.Arguments["tableNames"]
	if !ok {
		return mcp.NewToolResultError("テーブル名が指定されていません"), nil
	}

	// 配列への変換
	tableNamesInterface, ok := tableNamesRaw.([]interface{})
	if !ok || len(tableNamesInterface) == 0 {
		return mcp.NewToolResultError("テーブル名の配列が正しく指定されていません"), nil
	}

	// テーブル名を文字列の配列に変換
	var tableNames []string
	for _, v := range tableNamesInterface {
		if tableName, ok := v.(string); ok && tableName != "" {
			tableNames = append(tableNames, tableName)
		}
	}

	if len(tableNames) == 0 {
		return mcp.NewToolResultError("有効なテーブル名が指定されていません"), nil
	}

	var sb strings.Builder

	// すべてのテーブルに対して情報を取得
	for i, tableName := range tableNames {
		// 2つ目以降のテーブルの前に区切り線を追加
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}

		// テーブル情報の取得
		tables, err := fetchTablesWithComments(ctx, dbName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("テーブル情報の取得に失敗しました: %v", err)), nil
		}

		// 指定されたテーブルを探す
		var tableInfo TableInfo
		var tableFound bool
		for _, t := range tables {
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

		// 主キー情報の取得
		primaryKeys, err := fetchPrimaryKeys(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("主キー情報の取得に失敗しました: %v", err)), nil
		}

		// 一意キー情報の取得
		uniqueKeys, err := fetchUniqueKeys(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("一意キー情報の取得に失敗しました: %v", err)), nil
		}

		// 外部キー情報の取得
		foreignKeys, err := fetchForeignKeys(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("外部キー情報の取得に失敗しました: %v", err)), nil
		}

		// カラム情報の取得
		columns, err := fetchTableColumns(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("カラム情報の取得に失敗しました: %v", err)), nil
		}

		// インデックス情報の取得
		indexes, err := fetchTableIndexes(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("インデックス情報の取得に失敗しました: %v", err)), nil
		}

		// 結果の整形
		// テーブル基本情報
		sb.WriteString(fmt.Sprintf("# テーブル: %s", tableName))
		if tableInfo.Comment != "" {
			sb.WriteString(fmt.Sprintf(" - %s", tableInfo.Comment))
		}
		sb.WriteString("\n\n")

		// カラム情報
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

		// キー情報
		sb.WriteString("## キー情報\n")

		// 主キー情報
		if len(primaryKeys) > 0 {
			pkStr := strings.Join(primaryKeys, ", ")
			if len(primaryKeys) > 1 {
				pkStr = fmt.Sprintf("(%s)", pkStr)
			}
			sb.WriteString(fmt.Sprintf("[PK: %s]\n", pkStr))
		}

		// 一意キー情報
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

		// 外部キー情報
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

		// インデックス情報
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

// fetchTableColumns はテーブルのカラム情報を取得する関数
func fetchTableColumns(ctx context.Context, dbName string, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT 
			COLUMN_NAME, 
			COLUMN_TYPE, 
			IS_NULLABLE, 
			COLUMN_DEFAULT, 
			IFNULL(COLUMN_COMMENT, '') AS COLUMN_COMMENT
		FROM 
			INFORMATION_SCHEMA.COLUMNS 
		WHERE 
			TABLE_SCHEMA = ? 
			AND TABLE_NAME = ? 
		ORDER BY 
			ORDINAL_POSITION
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		if err := rows.Scan(&col.Name, &col.Type, &col.IsNullable, &col.Default, &col.Comment); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

// fetchTableIndexes はテーブルのインデックス情報を取得する関数
func fetchTableIndexes(ctx context.Context, dbName string, tableName string) ([]IndexInfo, error) {
	query := `
		SELECT 
			INDEX_NAME, 
			COLUMN_NAME,
			NON_UNIQUE 
		FROM 
			INFORMATION_SCHEMA.STATISTICS 
		WHERE 
			TABLE_SCHEMA = ? 
			AND TABLE_NAME = ? 
			AND INDEX_NAME != 'PRIMARY'
			AND INDEX_NAME NOT IN (
				SELECT CONSTRAINT_NAME 
				FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS 
				WHERE TABLE_SCHEMA = ? 
				AND TABLE_NAME = ? 
				AND CONSTRAINT_TYPE IN ('UNIQUE', 'FOREIGN KEY')
			)
		ORDER BY 
			INDEX_NAME, 
			SEQ_IN_INDEX
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*IndexInfo)
	for rows.Next() {
		var indexName, columnName string
		var nonUnique bool
		if err := rows.Scan(&indexName, &columnName, &nonUnique); err != nil {
			return nil, err
		}

		idx, exists := indexMap[indexName]
		if !exists {
			idx = &IndexInfo{
				Name:   indexName,
				Unique: !nonUnique,
			}
			indexMap[indexName] = idx
		}
		idx.Columns = append(idx.Columns, columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var indexes []IndexInfo
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	return indexes, nil
}
