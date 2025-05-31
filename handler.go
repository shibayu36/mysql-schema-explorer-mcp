package main

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"
)

// Handler struct implements the MCP handler
type Handler struct {
	db          *DB
	fixedDBName string
}

func NewHandler(db *DB, fixedDBName string) *Handler {
	return &Handler{db: db, fixedDBName: fixedDBName}
}

// ListTables returns summary information for all tables
func (h *Handler) ListTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Use fixed DB name if set, otherwise get from request
	dbName := h.fixedDBName
	if dbName == "" {
		dbNameRaw, ok := request.Params.Arguments["dbName"]
		if !ok {
			return mcp.NewToolResultError("Database name is not specified"), nil
		}

		dbName, ok = dbNameRaw.(string)
		if !ok || dbName == "" {
			return mcp.NewToolResultError("Database name is not specified correctly"), nil
		}
	}

	// Get table information
	tables, err := h.db.FetchAllTableSummaries(ctx, dbName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get table information: %v", err)), nil
	}

	// No tables found
	if len(tables) == 0 {
		return mcp.NewToolResultText("No tables exist in the database."), nil
	}

	// Create output
	var output bytes.Buffer
	{
		tmpl, err := template.New("listTables").Funcs(funcMap).Parse(listTablesTemplate)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse template: %v", err)), nil
		}

		if err := tmpl.Execute(&output, ListTablesData{
			DBName: dbName,
			Tables: tables,
		}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to execute template: %v", err)), nil
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}

// DescribeTables is a handler method that returns detailed information for the specified tables
func (h *Handler) DescribeTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Use fixed DB name if set, otherwise get from request
	dbName := h.fixedDBName
	if dbName == "" {
		dbNameRaw, ok := request.Params.Arguments["dbName"]
		if !ok {
			return mcp.NewToolResultError("Database name is not specified"), nil
		}

		dbName, ok = dbNameRaw.(string)
		if !ok || dbName == "" {
			return mcp.NewToolResultError("Database name is not specified correctly"), nil
		}
	}

	// Create list of table names
	tableNamesRaw, ok := request.Params.Arguments["tableNames"]
	if !ok {
		return mcp.NewToolResultError("Table names are not specified"), nil
	}
	tableNamesInterface, ok := tableNamesRaw.([]interface{})
	if !ok || len(tableNamesInterface) == 0 {
		return mcp.NewToolResultError("Array of table names is not specified correctly"), nil
	}
	var tableNames []string
	for _, v := range tableNamesInterface {
		if tableName, ok := v.(string); ok && tableName != "" {
			tableNames = append(tableNames, tableName)
		}
	}
	if len(tableNames) == 0 {
		return mcp.NewToolResultError("No valid table names are specified"), nil
	}

	allTables, err := h.db.FetchAllTableSummaries(ctx, dbName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get table information: %v", err)), nil
	}

	// Prepare output
	var output bytes.Buffer
	tmpl, err := template.New("describeTableDetail").Funcs(funcMap).Parse(describeTableDetailTemplate)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse template: %v", err)), nil
	}

	// Get information for all tables
	for i, tableName := range tableNames {
		// Add a separator line before the second and subsequent tables
		if i > 0 {
			output.WriteString("\n---\n\n")
		}

		// Find the specified table
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
			output.WriteString(fmt.Sprintf("# Table: %s\nTable not found\n", tableName))
			continue
		}

		// Get table detail information
		primaryKeys, err := h.db.FetchPrimaryKeys(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get primary key information: %v", err)), nil
		}

		uniqueKeys, err := h.db.FetchUniqueKeys(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get unique key information: %v", err)), nil
		}

		foreignKeys, err := h.db.FetchForeignKeys(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get foreign key information: %v", err)), nil
		}

		columns, err := h.db.FetchTableColumns(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get column information: %v", err)), nil
		}

		indexes, err := h.db.FetchTableIndexes(ctx, dbName, tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get index information: %v", err)), nil
		}

		// Create data to pass to the template
		tableDetail := TableDetail{
			Name:        tableName,
			Comment:     tableInfo.Comment,
			Columns:     columns,
			PrimaryKeys: primaryKeys,
			UniqueKeys:  uniqueKeys,
			ForeignKeys: foreignKeys,
			Indexes:     indexes,
		}

		// Execute the template and write to the buffer
		if err := tmpl.Execute(&output, tableDetail); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to execute template: %v", err)), nil
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}
