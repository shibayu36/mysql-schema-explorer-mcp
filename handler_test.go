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
- orders - 注文ヘッダー [PK: id] [FK: user_id -> users.id]
- products - 商品マスター [PK: product_code] [UK: (maker_code, internal_code)]
- users - ユーザー情報 [PK: id] [UK: email; (tenant_id, employee_id); username]
`
	textContent := result.Content[0].(mcp.TextContent).Text
	assert.Equal(t, expectedOutput, textContent, "Output content should match the expected format")
}

func TestDescribeTables(t *testing.T) {
	// --- Arrange ---
	dbConn := setupTestDB(t, "testdata/schema.sql") // テストDBとスキーマを準備

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
			// Name: "DescribeTables", // 必要に応じてツール名を設定
			Arguments: map[string]interface{}{
				"dbName":     testDBName,
				"tableNames": []interface{}{"users", "products", "order_items"}, // 複数テーブルを指定
			},
		},
	}

	expectedOutput := `# テーブル: users - ユーザー情報

## カラム
- id: int NOT NULL [ユーザーシステムID]
- email: varchar(255) NOT NULL [メールアドレス]
- username: varchar(255) NOT NULL [ユーザー名]
- tenant_id: int NOT NULL [テナントID]
- employee_id: int NOT NULL [従業員ID]

## キー情報
[PK: id]
[UK: email; (tenant_id, employee_id); username]

---

# テーブル: products - 商品マスター

## カラム
- product_code: varchar(50) NOT NULL [商品コード（主キー）]
- maker_code: varchar(50) NOT NULL [メーカーコード]
- internal_code: varchar(50) NOT NULL [社内商品コード]
- product_name: varchar(255) NULL [商品名]

## キー情報
[PK: product_code]
[UK: (maker_code, internal_code)]
[INDEX: (maker_code, product_name); product_name]

---

# テーブル: order_items - 注文明細

## カラム
- order_id: int NOT NULL [注文ID (FK)]
- item_seq: int NOT NULL [注文明細連番]
- product_maker: varchar(50) NOT NULL [商品メーカーコード (FK)]
- product_internal_code: varchar(50) NOT NULL [商品社内コード (FK)]
- quantity: int NOT NULL [数量]

## キー情報
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
