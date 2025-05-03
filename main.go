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
		log.Fatalf("Failed to load configuration: %v", err)
	}

	sqlDB, err := connectDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize DB layer and handler
	db := NewDB(sqlDB)
	handler := NewHandler(db)

	s := server.NewMCPServer(
		"mysql-schema-mcp",
		"1.0.0",
	)

	listTables := mcp.NewTool(
		"list_tables",
		mcp.WithDescription("Returns a list of table information in the specified MySQL database."),
		mcp.WithString("dbName",
			mcp.Required(),
			mcp.Description("The name of the database to retrieve information from."),
		),
	)

	s.AddTool(listTables, handler.ListTables)

	describeTables := mcp.NewTool(
		"describe_tables",
		mcp.WithDescription("Returns detailed information for the specified tables."),
		mcp.WithString("dbName",
			mcp.Required(),
			mcp.Description("The name of the database to retrieve information from."),
		),
		mcp.WithArray(
			"tableNames",
			mcp.Items(
				map[string]interface{}{
					"type": "string",
				},
			),
			mcp.Required(),
			mcp.Description("The names of the tables to retrieve detailed information for (multiple names can be specified)."),
		),
	)

	s.AddTool(describeTables, handler.DescribeTables)

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
		return DBConfig{}, fmt.Errorf("DB_USER environment variable is not set")
	}

	password := os.Getenv("DB_PASSWORD")

	return DBConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}, nil
}
