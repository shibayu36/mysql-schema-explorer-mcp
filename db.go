package main

import (
	"context"
	"database/sql"
	"fmt"
)

// DBConfig はデータベース接続設定を保持する構造体
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

// TableInfo はテーブル情報を保持する構造体
type TableInfo struct {
	Name    string
	Comment string
	PK      []string     // 主キーカラム
	UK      []UniqueKey  // 一意キー情報
	FK      []ForeignKey // 外部キー情報
}

// UniqueKey は一意キー情報を保持する構造体
type UniqueKey struct {
	Name    string
	Columns []string
}

// ForeignKey は外部キー情報を保持する構造体
type ForeignKey struct {
	Name       string
	Columns    []string
	RefTable   string
	RefColumns []string
}

// ColumnInfo はカラム情報を保持する構造体
type ColumnInfo struct {
	Name       string
	Type       string
	IsNullable string
	Default    sql.NullString
	Comment    string
}

// IndexInfo はインデックス情報を保持する構造体
type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
}

// グローバル変数
var db *sql.DB

func connectDB(config DBConfig) (*sql.DB, error) {
	// データベース名を指定せずに接続（各ツール実行時にデータベースを指定する）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/",
		config.User, config.Password, config.Host, config.Port)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// fetchTablesWithAllInfo はテーブル名、コメント、および全てのキー情報を取得する関数
func fetchTablesWithAllInfo(ctx context.Context, dbName string) ([]TableInfo, error) {
	// 基本的なテーブル情報を取得
	tables, err := fetchTablesWithComments(ctx, dbName)
	if err != nil {
		return nil, err
	}

	// 各テーブルの追加情報を取得
	for i := range tables {
		// 主キー情報を取得
		tables[i].PK, err = fetchPrimaryKeys(ctx, dbName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		// 一意キー情報を取得
		tables[i].UK, err = fetchUniqueKeys(ctx, dbName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		// 外部キー情報を取得
		tables[i].FK, err = fetchForeignKeys(ctx, dbName, tables[i].Name)
		if err != nil {
			return nil, err
		}
	}

	return tables, nil
}

// fetchTablesWithComments はテーブル名とコメントを取得する関数
func fetchTablesWithComments(ctx context.Context, dbName string) ([]TableInfo, error) {
	query := `
		SELECT 
			TABLE_NAME, 
			IFNULL(TABLE_COMMENT, '') AS TABLE_COMMENT 
		FROM 
			INFORMATION_SCHEMA.TABLES 
		WHERE 
			TABLE_SCHEMA = ? 
		ORDER BY 
			TABLE_NAME
	`

	rows, err := db.QueryContext(ctx, query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		if err := rows.Scan(&table.Name, &table.Comment); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

// fetchPrimaryKeys はテーブルの主キーカラムを取得する関数
func fetchPrimaryKeys(ctx context.Context, dbName string, tableName string) ([]string, error) {
	query := `
		SELECT 
			COLUMN_NAME
		FROM 
			INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
		WHERE 
			CONSTRAINT_SCHEMA = ? 
			AND TABLE_NAME = ? 
			AND CONSTRAINT_NAME = 'PRIMARY'
		ORDER BY 
			ORDINAL_POSITION
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return primaryKeys, nil
}

// fetchUniqueKeys はテーブルの一意キー制約を取得する関数
func fetchUniqueKeys(ctx context.Context, dbName string, tableName string) ([]UniqueKey, error) {
	query := `
		SELECT 
			kcu.CONSTRAINT_NAME,
			kcu.COLUMN_NAME
		FROM 
			INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
		JOIN 
			INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
		ON 
			kcu.CONSTRAINT_SCHEMA = tc.CONSTRAINT_SCHEMA
			AND kcu.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
			AND kcu.TABLE_NAME = tc.TABLE_NAME
		WHERE 
			kcu.TABLE_SCHEMA = ? 
			AND kcu.TABLE_NAME = ? 
			AND tc.CONSTRAINT_TYPE = 'UNIQUE'
		ORDER BY 
			kcu.CONSTRAINT_NAME,
			kcu.ORDINAL_POSITION
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ukMap := make(map[string][]string)
	for rows.Next() {
		var constraintName, columnName string
		if err := rows.Scan(&constraintName, &columnName); err != nil {
			return nil, err
		}
		ukMap[constraintName] = append(ukMap[constraintName], columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var uniqueKeys []UniqueKey
	for name, columns := range ukMap {
		uniqueKeys = append(uniqueKeys, UniqueKey{
			Name:    name,
			Columns: columns,
		})
	}

	return uniqueKeys, nil
}

// fetchForeignKeys はテーブルの外部キー制約を取得する関数
func fetchForeignKeys(ctx context.Context, dbName string, tableName string) ([]ForeignKey, error) {
	query := `
		SELECT 
			kcu.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME
		FROM 
			INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
		JOIN 
			INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS rc
		ON 
			kcu.CONSTRAINT_SCHEMA = rc.CONSTRAINT_SCHEMA
			AND kcu.CONSTRAINT_NAME = rc.CONSTRAINT_NAME
		WHERE 
			kcu.TABLE_SCHEMA = ? 
			AND kcu.TABLE_NAME = ? 
			AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY 
			kcu.CONSTRAINT_NAME,
			kcu.ORDINAL_POSITION
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[string]ForeignKey)
	for rows.Next() {
		var constraintName, columnName, refTableName, refColumnName string
		if err := rows.Scan(&constraintName, &columnName, &refTableName, &refColumnName); err != nil {
			return nil, err
		}

		fk, exists := fkMap[constraintName]
		if !exists {
			fk = ForeignKey{
				Name:     constraintName,
				RefTable: refTableName,
			}
		}

		fk.Columns = append(fk.Columns, columnName)
		fk.RefColumns = append(fk.RefColumns, refColumnName)
		fkMap[constraintName] = fk
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var foreignKeys []ForeignKey
	for _, fk := range fkMap {
		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

// fetchTableColumns はテーブルのカラム情報を取得する関数
func fetchTableColumns(ctx context.Context, dbName string, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT 
			COLUMN_NAME, 
			COLUMN_TYPE, 
			IS_NULLABLE, 
			COLUMN_DEFAULT, 
			IFNULL(COLUMN_COMMENT, '') AS COLUMN_COMMENT
		FROM 
			INFORMATION_SCHEMA.COLUMNS 
		WHERE 
			TABLE_SCHEMA = ? 
			AND TABLE_NAME = ? 
		ORDER BY 
			ORDINAL_POSITION
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		if err := rows.Scan(&col.Name, &col.Type, &col.IsNullable, &col.Default, &col.Comment); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

// fetchTableIndexes はテーブルのインデックス情報を取得する関数
func fetchTableIndexes(ctx context.Context, dbName string, tableName string) ([]IndexInfo, error) {
	query := `
		SELECT 
			INDEX_NAME, 
			COLUMN_NAME,
			NON_UNIQUE 
		FROM 
			INFORMATION_SCHEMA.STATISTICS 
		WHERE 
			TABLE_SCHEMA = ? 
			AND TABLE_NAME = ? 
			AND INDEX_NAME != 'PRIMARY'
			AND INDEX_NAME NOT IN (
				SELECT CONSTRAINT_NAME 
				FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS 
				WHERE TABLE_SCHEMA = ? 
				AND TABLE_NAME = ? 
				AND CONSTRAINT_TYPE IN ('UNIQUE', 'FOREIGN KEY')
			)
		ORDER BY 
			INDEX_NAME, 
			SEQ_IN_INDEX
	`

	rows, err := db.QueryContext(ctx, query, dbName, tableName, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*IndexInfo)
	for rows.Next() {
		var indexName, columnName string
		var nonUnique bool
		if err := rows.Scan(&indexName, &columnName, &nonUnique); err != nil {
			return nil, err
		}

		idx, exists := indexMap[indexName]
		if !exists {
			idx = &IndexInfo{
				Name:   indexName,
				Unique: !nonUnique,
			}
			indexMap[indexName] = idx
		}
		idx.Columns = append(idx.Columns, columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var indexes []IndexInfo
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	return indexes, nil
}
