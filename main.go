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
	tables, err := fetchTablesWithComments(ctx)
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
	sb.WriteString("フォーマット: テーブル名 - テーブルコメント\n\n")

	// テーブルリスト
	for _, table := range tables {
		comment := table.Comment
		if comment == "" {
			comment = "(コメントなし)"
		}
		sb.WriteString(fmt.Sprintf("- %s - %s\n", table.Name, comment))
	}

	return mcp.NewToolResultText(sb.String()), nil
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
