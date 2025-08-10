package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const Version = "1.1.1"

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
	fixedDBName := os.Getenv("DB_NAME")
	handler := NewHandler(db, fixedDBName)

	s := server.NewMCPServer(
		"mysql-schema-mcp",
		Version,
	)

	// Build list_tables tool options
	listTablesOpts := []mcp.ToolOption{
		mcp.WithDescription("Returns a list of table information in the MySQL database."),
	}
	if fixedDBName == "" {
		listTablesOpts = append(listTablesOpts, mcp.WithString("dbName",
			mcp.Required(),
			mcp.Description("The name of the database to retrieve information from."),
		))
	}
	s.AddTool(
		mcp.NewTool("list_tables", listTablesOpts...),
		handler.ListTables,
	)

	// Build describe_tables tool options
	describeTablesOpts := []mcp.ToolOption{
		mcp.WithDescription("Returns detailed information for the specified tables."),
	}
	if fixedDBName == "" {
		describeTablesOpts = append(describeTablesOpts, mcp.WithString("dbName",
			mcp.Required(),
			mcp.Description("The name of the database to retrieve information from."),
		))
	}
	describeTablesOpts = append(describeTablesOpts, mcp.WithArray(
		"tableNames",
		mcp.Items(
			map[string]interface{}{
				"type": "string",
			},
		),
		mcp.Required(),
		mcp.Description("The names of the tables to retrieve detailed information for (multiple names can be specified)."),
	))
	s.AddTool(
		mcp.NewTool("describe_tables", describeTablesOpts...),
		handler.DescribeTables,
	)

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
