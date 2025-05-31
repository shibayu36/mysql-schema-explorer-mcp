# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -run TestListTables
```

### Building
```bash
# Build the binary
go build -o mysql-schema-explorer-mcp

# Install globally
go install
```

### Local Development with Docker
```bash
# Start MySQL test instance
docker-compose up -d

# Set environment variables for local testing
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=root
```

## Architecture

This is a Model Context Protocol (MCP) server that provides MySQL schema information to LLM applications. The codebase follows a clean architecture pattern:

- **main.go**: Entry point that sets up the MCP server, loads DB configuration from environment variables, and registers the two available tools (list_tables and describe_tables)
- **handler.go**: Contains the business logic for handling MCP tool requests. Implements ListTables and DescribeTables methods
- **db.go**: Database layer that handles MySQL connections and queries. Provides methods for fetching table metadata, columns, indexes, and constraints
- **view.go**: Formatting layer that converts raw database schema information into human-readable output

The server uses the mcp-go library (github.com/mark3labs/mcp-go) to implement the MCP protocol and communicates via stdio.

## MCP Tool Implementation

When implementing new tools:
1. Define the tool in main.go using mcp.NewTool() with proper descriptions and parameters
2. Add the handler method to handler.go that processes the request
3. Implement any required database queries in db.go
4. Format the output in view.go if needed

Tools must handle errors gracefully and return meaningful error messages through the MCP protocol.

## Makefile Preferences

When writing Makefiles:
- Place `.PHONY` declarations immediately before each target rather than grouping them at the top
- Example:
  ```makefile
  .PHONY: fmt
  fmt:
  	go fmt ./...
  
  .PHONY: test
  test:
  	go test ./...
  ```