package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type jsonRPCRequest struct {
	ID     interface{}            `json:"id,omitempty"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	ID     interface{}            `json:"id"`
	Result json.RawMessage        `json:"result,omitempty"`
	Error  map[string]interface{} `json:"error,omitempty"`
}

type mcpServer struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
	nextID int
}

func setupMCPServer(t *testing.T, env []string) *mcpServer {
	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(), env...)

	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	server := &mcpServer{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		reader: bufio.NewReader(stdout),
		nextID: 1,
	}

	t.Cleanup(func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	return server
}

func initializeMCPServer(t *testing.T, server *mcpServer) {
	// Send initialize request
	initReq := jsonRPCRequest{
		Method: "initialize",
	}
	server.sendRequest(t, initReq)

	// Read initialize response
	initResp := server.readResponse(t)
	require.Empty(t, initResp.Error, "Initialize should succeed")

	// Send initialized notification
	initializedReq := jsonRPCRequest{
		Method: "notifications/initialized",
	}
	server.sendRequest(t, initializedReq)
}

func (s *mcpServer) sendRequest(t *testing.T, req jsonRPCRequest) {
	// Auto-increment ID for requests (except notifications)
	if req.Method != "notifications/initialized" && req.ID == nil {
		req.ID = s.nextID
		s.nextID++
	}

	// Convert to the actual JSON-RPC format with jsonrpc field
	fullReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  req.Method,
	}
	if req.ID != nil {
		fullReq["id"] = req.ID
	}
	if req.Params != nil {
		fullReq["params"] = req.Params
	}

	data, err := json.Marshal(fullReq)
	require.NoError(t, err)

	_, err = fmt.Fprintf(s.stdin, "%s\n", data)
	require.NoError(t, err)
}

func (s *mcpServer) readResponse(t *testing.T) jsonRPCResponse {
	line, err := s.reader.ReadBytes('\n')
	require.NoError(t, err)

	var resp jsonRPCResponse
	err = json.Unmarshal(line, &resp)
	require.NoError(t, err)

	return resp
}

// Common test setup helper
func setupE2ETest(t *testing.T) *mcpServer {
	config := createTestDBConfig(t)
	_ = setupTestDB(t, "testdata/schema.sql")

	env := []string{
		fmt.Sprintf("DB_HOST=%s", config.Host),
		fmt.Sprintf("DB_PORT=%s", config.Port),
		fmt.Sprintf("DB_USER=%s", config.User),
		fmt.Sprintf("DB_PASSWORD=%s", config.Password),
	}

	server := setupMCPServer(t, env)
	initializeMCPServer(t, server)
	return server
}

// Helper to send tools/call request
func (s *mcpServer) sendToolCallRequest(t *testing.T, toolName string, arguments map[string]interface{}) {
	req := jsonRPCRequest{
		Method: "tools/call",
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}
	s.sendRequest(t, req)
}

// Helper to verify response and extract text content
func verifyTextResponse(t *testing.T, resp jsonRPCResponse) string {
	// Check no error
	assert.Empty(t, resp.Error)

	// Parse result
	var result map[string]interface{}
	err := json.Unmarshal(resp.Result, &result)
	require.NoError(t, err)

	// Verify content
	content, ok := result["content"].([]interface{})
	require.True(t, ok)
	require.Len(t, content, 1)

	textContent := content[0].(map[string]interface{})
	assert.Equal(t, "text", textContent["type"])

	text := textContent["text"].(string)
	return text
}

func TestE2EListTables(t *testing.T) {
	server := setupE2ETest(t)

	// Send list_tables request
	server.sendToolCallRequest(t, "list_tables", map[string]interface{}{
		"dbName": testDBName,
	})

	// Read and verify response
	resp := server.readResponse(t)
	text := verifyTextResponse(t, resp)

	expectedText := `Tables in database "test_mysql_schema_explorer_mcp" (Total: 4)
Format: Table Name - Table Comment [PK: Primary Key] [UK: Unique Key 1; Unique Key 2...] [FK: Foreign Key -> Referenced Table.Column; ...]
* Composite keys (keys composed of multiple columns) are grouped in parentheses: (col1, col2)
* Multiple different key constraints are separated by semicolons: key1; key2

- order_items - Order details [PK: (order_id, item_seq)] [UK: (order_id, product_maker, product_internal_code)] [FK: order_id -> orders.id; (product_maker, product_internal_code) -> products.(maker_code, internal_code)]
- orders - Order header [PK: id] [FK: user_id -> users.id]
- products - Product master [PK: product_code] [UK: (maker_code, internal_code)]
- users - User information [PK: id] [UK: email; (tenant_id, employee_id); username]
`

	assert.Equal(t, expectedText, text)
}

func TestE2EDescribeTables(t *testing.T) {
	server := setupE2ETest(t)

	// Send describe_tables request
	server.sendToolCallRequest(t, "describe_tables", map[string]interface{}{
		"dbName":     testDBName,
		"tableNames": []string{"users", "products", "order_items"},
	})

	// Read and verify response
	resp := server.readResponse(t)
	text := verifyTextResponse(t, resp)

	expectedText := `# Table: users - User information

## Columns
- id: int NOT NULL [User system ID]
- email: varchar(255) NOT NULL [Email address]
- username: varchar(255) NOT NULL [Username]
- tenant_id: int NOT NULL [Tenant ID]
- employee_id: int NOT NULL [Employee ID]

## Key Information
[PK: id]
[UK: email; (tenant_id, employee_id); username]

---

# Table: products - Product master

## Columns
- product_code: varchar(50) NOT NULL [Product code (Primary Key)]
- maker_code: varchar(50) NOT NULL [Maker code]
- internal_code: varchar(50) NOT NULL [Internal product code]
- product_name: varchar(255) NULL [Product name]

## Key Information
[PK: product_code]
[UK: (maker_code, internal_code)]
[INDEX: (maker_code, product_name); product_name]

---

# Table: order_items - Order details

## Columns
- order_id: int NOT NULL [Order ID (FK)]
- item_seq: int NOT NULL [Order item sequence number]
- product_maker: varchar(50) NOT NULL [Product maker code (FK)]
- product_internal_code: varchar(50) NOT NULL [Product internal code (FK)]
- quantity: int NOT NULL [Quantity]

## Key Information
[PK: (order_id, item_seq)]
[UK: (order_id, product_maker, product_internal_code)]
[FK: order_id -> orders.id; (product_maker, product_internal_code) -> products.(maker_code, internal_code)]
[INDEX: (product_maker, product_internal_code)]
`

	assert.Equal(t, expectedText, text)
}

func TestE2E_FixedDBMode(t *testing.T) {
	config := createTestDBConfig(t)
	_ = setupTestDB(t, "testdata/schema.sql")

	// Setup with DB_NAME environment variable
	env := []string{
		fmt.Sprintf("DB_HOST=%s", config.Host),
		fmt.Sprintf("DB_PORT=%s", config.Port),
		fmt.Sprintf("DB_USER=%s", config.User),
		fmt.Sprintf("DB_PASSWORD=%s", config.Password),
		fmt.Sprintf("DB_NAME=%s", testDBName), // Fixed DB name
	}

	server := setupMCPServer(t, env)
	initializeMCPServer(t, server)

	t.Run("list_tables works without dbName in fixed mode", func(t *testing.T) {
		// Send without dbName parameter
		server.sendToolCallRequest(t, "list_tables", map[string]interface{}{})

		resp := server.readResponse(t)
		text := verifyTextResponse(t, resp)

		// Should list tables from the fixed DB
		assert.Contains(t, text, "Tables in database \"test_mysql_schema_explorer_mcp\" (Total: 4)")
		assert.Contains(t, text, "users")
		assert.Contains(t, text, "orders")
		assert.Contains(t, text, "products")
		assert.Contains(t, text, "order_items")
	})

	t.Run("describe_tables works without dbName in fixed mode", func(t *testing.T) {
		// Send without dbName parameter
		server.sendToolCallRequest(t, "describe_tables", map[string]interface{}{
			"tableNames": []string{"users"},
		})

		resp := server.readResponse(t)
		text := verifyTextResponse(t, resp)

		// Should describe table from the fixed DB
		assert.Contains(t, text, "# Table: users")
		assert.Contains(t, text, "## Columns")
		assert.Contains(t, text, "- id: int NOT NULL")
		assert.Contains(t, text, "- email: varchar(255) NOT NULL")
		assert.Contains(t, text, "- username: varchar(255) NOT NULL")
	})

}

func TestE2E_ToolDefinitions(t *testing.T) {
	findTool := func(tools []interface{}, name string) map[string]interface{} {
		for _, tool := range tools {
			toolMap := tool.(map[string]interface{})
			if toolMap["name"] == name {
				return toolMap
			}
		}
		return nil
	}

	getToolProperties := func(tool map[string]interface{}) map[string]interface{} {
		inputSchema, ok := tool["inputSchema"]
		if !ok || inputSchema == nil {
			return map[string]interface{}{}
		}

		properties, ok := inputSchema.(map[string]interface{})["properties"]
		if !ok || properties == nil {
			return map[string]interface{}{}
		}

		return properties.(map[string]interface{})
	}

	t.Run("normal mode has dbName parameter", func(t *testing.T) {
		server := setupE2ETest(t) // Normal mode (no DB_NAME)

		req := jsonRPCRequest{Method: "tools/list"}
		server.sendRequest(t, req)
		resp := server.readResponse(t)

		var result map[string]interface{}
		json.Unmarshal(resp.Result, &result)
		tools := result["tools"].([]interface{})

		assert.Len(t, tools, 2)

		// Check list_tables has dbName parameter
		listTables := findTool(tools, "list_tables")
		properties := getToolProperties(listTables)
		_, hasDBName := properties["dbName"]
		assert.True(t, hasDBName, "list_tables should have dbName in normal mode")

		// Check describe_tables has dbName parameter
		describeTables := findTool(tools, "describe_tables")
		properties = getToolProperties(describeTables)
		_, hasDBName = properties["dbName"]
		assert.True(t, hasDBName, "describe_tables should have dbName in normal mode")
	})

	t.Run("fixed mode has no dbName parameter", func(t *testing.T) {
		config := createTestDBConfig(t)
		_ = setupTestDB(t, "testdata/schema.sql")

		env := []string{
			fmt.Sprintf("DB_HOST=%s", config.Host),
			fmt.Sprintf("DB_PORT=%s", config.Port),
			fmt.Sprintf("DB_USER=%s", config.User),
			fmt.Sprintf("DB_PASSWORD=%s", config.Password),
			fmt.Sprintf("DB_NAME=%s", testDBName),
		}

		server := setupMCPServer(t, env)
		initializeMCPServer(t, server)

		req := jsonRPCRequest{Method: "tools/list"}
		server.sendRequest(t, req)
		resp := server.readResponse(t)

		var result map[string]interface{}
		json.Unmarshal(resp.Result, &result)
		tools := result["tools"].([]interface{})

		assert.Len(t, tools, 2)

		// Check list_tables has no dbName parameter
		listTables := findTool(tools, "list_tables")
		properties := getToolProperties(listTables)
		_, hasDBName := properties["dbName"]
		assert.False(t, hasDBName, "list_tables should not have dbName in fixed mode")

		// Check describe_tables has no dbName parameter
		describeTables := findTool(tools, "describe_tables")
		properties = getToolProperties(describeTables)
		_, hasDBName = properties["dbName"]
		assert.False(t, hasDBName, "describe_tables should not have dbName in fixed mode")
	})
}
