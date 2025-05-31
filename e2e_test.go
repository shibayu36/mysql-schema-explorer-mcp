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
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Result  json.RawMessage        `json:"result,omitempty"`
	Error   map[string]interface{} `json:"error,omitempty"`
}

type mcpServer struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
}

func startMCPServer(t *testing.T, env []string) *mcpServer {
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
	}
	
	t.Cleanup(func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})
	
	// Wait for server to be ready and initialize
	time.Sleep(100 * time.Millisecond)
	
	// Send initialize request
	initReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "0.1.0",
			"capabilities": map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
	server.sendRequest(t, initReq)
	
	// Read initialize response
	initResp := server.readResponse(t)
	require.Empty(t, initResp.Error)
	
	// Send initialized notification
	initializedReq := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	server.sendRequest(t, initializedReq)
	
	return server
}

func (s *mcpServer) sendRequest(t *testing.T, req jsonRPCRequest) {
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}
	data, err := json.Marshal(req)
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

func TestE2EListTables(t *testing.T) {
	// Setup test database
	config := createTestDBConfig(t)
	_ = setupTestDB(t, "testdata/schema.sql")
	
	// Start MCP server with test DB configuration
	env := []string{
		fmt.Sprintf("DB_HOST=%s", config.Host),
		fmt.Sprintf("DB_PORT=%s", config.Port),
		fmt.Sprintf("DB_USER=%s", config.User),
		fmt.Sprintf("DB_PASSWORD=%s", config.Password),
	}
	
	server := startMCPServer(t, env)
	
	// Send list_tables request
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "list_tables",
			"arguments": map[string]interface{}{
				"dbName": testDBName,
			},
		},
	}
	
	server.sendRequest(t, req)
	
	// Read and verify response
	resp := server.readResponse(t)
	
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