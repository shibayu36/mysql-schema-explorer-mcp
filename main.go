package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"mysql-schema-mcp",
		"1.0.0",
	)

	listTables := mcp.NewTool(
		"list_tables",
		mcp.WithDescription("MySQLのデータベース内のテーブル情報を一覧で返す"),
	)

	s.AddTool(listTables, listTablesHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func listTablesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("tables"), nil
}
