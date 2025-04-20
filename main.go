package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

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
	return mcp.NewToolResultText("tables"), nil
}
