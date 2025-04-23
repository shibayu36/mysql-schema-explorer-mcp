package main

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTables(t *testing.T) {
	// --- Setup ---
	dbConn, cleanup := setupTestDB(t)
	defer cleanup()

	applySchema(t, dbConn, "testdata/schema.sql")

	db := NewDB(dbConn)
	handler := NewHandler(db)

	ctx := context.Background()
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
				"dbName": "test_db",
			},
		},
	}

	// --- Execute ---
	result, err := handler.ListTables(ctx, req)

	// --- Assert ---
	require.NoError(t, err, "handler.ListTables should not return an error")
	require.NotNil(t, result, "handler.ListTables should return a result")

	// ResultType のチェックは mcp-go の仕様に合わせて削除 (エラーから存在しないと判断)
	// assert.Equal(t, mcp.ResultTypeText, result.ResultType, "ResultType should be text")

	// Expected output from docs/テスト方針.md
	expectedOutput := `データベース「test_db」のテーブル一覧 (全4件)
フォーマット: テーブル名 - テーブルコメント [PK: 主キー] [UK: 一意キー1; 一意キー2...] [FK: 外部キー -> 参照先テーブル.カラム; ...]
※ 複合キー（複数カラムで構成されるキー）は括弧でグループ化: (col1, col2)
※ 複数の異なるキー制約はセミコロンで区切り: key1; key2

- order_items - 注文明細 [PK: (order_id, item_seq)] [UK: (order_id, product_maker, product_internal_code)] [FK: order_id -> orders.id; (product_maker, product_internal_code) -> products.(maker_code, internal_code)]
- orders - 注文ヘッダー [PK: id]
- products - 商品マスター [PK: product_code] [UK: (maker_code, internal_code)]
- users - ユーザー情報 [PK: id] [UK: email; (tenant_id, employee_id); username]
`
	// Note: 行末の空白や改行コードの違いを吸収するため、比較前に標準化するなどの工夫が必要な場合がある
	// ここでは単純比較を行う
	textContent := result.Content[0].(mcp.TextContent).Text
	assert.Equal(t, expectedOutput, textContent, "Output content should match the expected format")
}
