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
	DBName   string
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
	)

	s.AddTool(listTables, listTablesHandler)

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

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return DBConfig{}, fmt.Errorf("DB_NAME環境変数が設定されていません")
	}

	return DBConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbName,
	}, nil
}

func connectDB(config DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.User, config.Password, config.Host, config.Port, config.DBName)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func listTablesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// テーブル情報の取得
	tables, err := fetchTablesWithAllInfo(ctx)
	if err != nil {
		// エラーが発生した場合は適切なエラーメッセージを返す
		return mcp.NewToolResultError(fmt.Sprintf("テーブル情報の取得に失敗しました: %v", err)), nil
	}

	// テーブルが見つからない場合
	if len(tables) == 0 {
		return mcp.NewToolResultText("データベース内にテーブルが存在しません。"), nil
	}

	// データベース名
	dbName := os.Getenv("DB_NAME")

	// フォーマット済みのテキスト出力を構築
	var sb strings.Builder

	// ヘッダー部分
	sb.WriteString(fmt.Sprintf("データベース「%s」のテーブル一覧 (全%d件)\n", dbName, len(tables)))
	sb.WriteString("フォーマット: テーブル名 - テーブルコメント [PK: 主キー] [UK: 一意キー1, 一意キー2...] [FK: 外部キー -> 参照先テーブル.カラム, ...]\n\n")

	// テーブルリスト
	for _, table := range tables {
		// 基本情報
		sb.WriteString(fmt.Sprintf("- %s - %s", table.Name, table.Comment))

		// 主キー情報
		if len(table.PK) > 0 {
			sb.WriteString(fmt.Sprintf(" [PK: %s]", strings.Join(table.PK, ", ")))
		}

		// 一意キー情報
		if len(table.UK) > 0 {
			var ukInfo []string
			for _, uk := range table.UK {
				ukInfo = append(ukInfo, strings.Join(uk.Columns, ", "))
			}
			sb.WriteString(fmt.Sprintf(" [UK: %s]", strings.Join(ukInfo, ", ")))
		}

		// 外部キー情報
		if len(table.FK) > 0 {
			var fkInfo []string
			for _, fk := range table.FK {
				fkInfo = append(fkInfo, fmt.Sprintf("%s -> %s.%s",
					strings.Join(fk.Columns, ", "),
					fk.RefTable,
					strings.Join(fk.RefColumns, ", ")))
			}
			sb.WriteString(fmt.Sprintf(" [FK: %s]", strings.Join(fkInfo, ", ")))
		}

		sb.WriteString("\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// fetchTablesWithAllInfo はテーブル名、コメント、および全てのキー情報を取得する関数
func fetchTablesWithAllInfo(ctx context.Context) ([]TableInfo, error) {
	// 基本的なテーブル情報を取得
	tables, err := fetchTablesWithComments(ctx)
	if err != nil {
		return nil, err
	}

	// 各テーブルの追加情報を取得
	dbName := os.Getenv("DB_NAME")
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
func fetchTablesWithComments(ctx context.Context) ([]TableInfo, error) {
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

	rows, err := db.QueryContext(ctx, query, os.Getenv("DB_NAME"))
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
