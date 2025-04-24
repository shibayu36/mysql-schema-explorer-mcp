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
			// Name: "ListTables", // 必要に応じてツール名を設定
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

	expectedOutput := `データベース「` + testDBName + `」のテーブル一覧 (全4件)
フォーマット: テーブル名 - テーブルコメント [PK: 主キー] [UK: 一意キー1; 一意キー2...] [FK: 外部キー -> 参照先テーブル.カラム; ...]
※ 複合キー（複数カラムで構成されるキー）は括弧でグループ化: (col1, col2)
※ 複数の異なるキー制約はセミコロンで区切り: key1; key2

- order_items - 注文明細 [PK: (order_id, item_seq)] [UK: (order_id, product_maker, product_internal_code)] [FK: order_id -> orders.id; (product_maker, product_internal_code) -> products.(maker_code, internal_code)]
- orders - 注文ヘッダー [PK: id]
- products - 商品マスター [PK: product_code] [UK: (maker_code, internal_code)]
- users - ユーザー情報 [PK: id] [UK: email; (tenant_id, employee_id); username]
`
	textContent := result.Content[0].(mcp.TextContent).Text
	assert.Equal(t, expectedOutput, textContent, "Output content should match the expected format")
}
