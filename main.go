package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

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
