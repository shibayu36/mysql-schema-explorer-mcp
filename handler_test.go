package main

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTables(t *testing.T) {
	dbConn := setupTestDB(t, "testdata/schema.sql")

	db := NewDB(dbConn)
	handler := NewHandler(db)

	ctx := t.Context()
	req := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			// Name: "ListTables", // Set tool name if necessary
			Arguments: map[string]interface{}{
				"dbName": testDBName,
			},
		},
	}

	// --- Act ---
	result, err := handler.ListTables(ctx, req)

	// --- Assert ---
	require.NoError(t, err, "handler.ListTables should not return an error")
	require.NotNil(t, result, "handler.ListTables should return a result")

	expectedOutput := `Tables in database "` + testDBName + `" (Total: 4)
Format: Table Name - Table Comment [PK: Primary Key] [UK: Unique Key 1; Unique Key 2...] [FK: Foreign Key -> Referenced Table.Column; ...]
* Composite keys (keys composed of multiple columns) are grouped in parentheses: (col1, col2)
* Multiple different key constraints are separated by semicolons: key1; key2

- order_items - Order details [PK: (order_id, item_seq)] [UK: (order_id, product_maker, product_internal_code)] [FK: order_id -> orders.id; (product_maker, product_internal_code) -> products.(maker_code, internal_code)]
- orders - Order header [PK: id] [FK: user_id -> users.id]
- products - Product master [PK: product_code] [UK: (maker_code, internal_code)]
- users - User information [PK: id] [UK: email; (tenant_id, employee_id); username]
`
	textContent := result.Content[0].(mcp.TextContent).Text
	assert.Equal(t, expectedOutput, textContent, "Output content should match the expected format")
}

func TestDescribeTables(t *testing.T) {
	// --- Arrange ---
	dbConn := setupTestDB(t, "testdata/schema.sql") // Prepare test DB and schema

	db := NewDB(dbConn)
	handler := NewHandler(db)

	ctx := t.Context()
	req := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			// Name: "DescribeTables", // Set tool name if necessary
			Arguments: map[string]interface{}{
				"dbName":     testDBName,
				"tableNames": []interface{}{"users", "products", "order_items"}, // Specify multiple tables
			},
		},
	}

	expectedOutput := `# Table: users - User information

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

	// --- Act ---
	result, err := handler.DescribeTables(ctx, req)

	// --- Assert ---
	require.NoError(t, err, "handler.DescribeTables should not return an error")
	require.NotNil(t, result, "handler.DescribeTables should return a result")
	require.Len(t, result.Content, 1, "Result should contain one content item")
	require.IsType(t, mcp.TextContent{}, result.Content[0], "Content item should be TextContent")

	textContent := result.Content[0].(mcp.TextContent).Text

	assert.Equal(t, expectedOutput, textContent, "Output content should match the expected format")
}
