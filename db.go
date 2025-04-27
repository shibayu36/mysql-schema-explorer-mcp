package main

import (
	"context"
	"database/sql"
	"fmt"
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

type TableSummary struct {
	Name    string
	Comment string
	PK      []string     // 主キーカラム
	UK      []UniqueKey  // 一意キー情報
	FK      []ForeignKey // 外部キー情報
}

type UniqueKey struct {
	Name    string
	Columns []string
}

type ForeignKey struct {
	Name       string
	Columns    []string
	RefTable   string
	RefColumns []string
}

type ColumnInfo struct {
	Name       string
	Type       string
	IsNullable string
	Default    sql.NullString
	Comment    string
}

type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
}

type DB struct {
	conn *sql.DB
}

func NewDB(conn *sql.DB) *DB {
	return &DB{conn: conn}
}

func connectDB(config DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/",
		config.User, config.Password, config.Host, config.Port)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// FetchAllTableSummaries データベース内の全てのテーブルのサマリー情報を取得
func (db *DB) FetchAllTableSummaries(ctx context.Context, dbName string) ([]TableSummary, error) {
	tables, err := db.FetchTableWithComments(ctx, dbName)
	if err != nil {
		return nil, err
	}

	// 各テーブルの追加情報を取得
	for i := range tables {
		tables[i].PK, err = db.FetchPrimaryKeys(ctx, dbName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		tables[i].UK, err = db.FetchUniqueKeys(ctx, dbName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		tables[i].FK, err = db.FetchForeignKeys(ctx, dbName, tables[i].Name)
		if err != nil {
			return nil, err
		}
	}

	return tables, nil
}

// FetchTableWithComments テーブル名とコメントを取得
func (db *DB) FetchTableWithComments(ctx context.Context, dbName string) ([]TableSummary, error) {
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

	rows, err := db.conn.QueryContext(ctx, query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableSummary
	for rows.Next() {
		var table TableSummary
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

// FetchPrimaryKeys テーブルの主キーカラムを取得
func (db *DB) FetchPrimaryKeys(ctx context.Context, dbName string, tableName string) ([]string, error) {
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

	rows, err := db.conn.QueryContext(ctx, query, dbName, tableName)
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

// FetchUniqueKeys テーブルの一意キー制約を取得
func (db *DB) FetchUniqueKeys(ctx context.Context, dbName string, tableName string) ([]UniqueKey, error) {
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

	rows, err := db.conn.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// SQL取得の順序を維持しながら情報を構築する
	var uniqueKeys []UniqueKey
	var currentUniqueKey *UniqueKey
	for rows.Next() {
		var constraintName, columnName string
		if err := rows.Scan(&constraintName, &columnName); err != nil {
			return nil, err
		}

		if currentUniqueKey == nil || currentUniqueKey.Name != constraintName {
			// 一つ目、もしくは別のUKに切り替わった時
			newUK := UniqueKey{
				Name:    constraintName,
				Columns: []string{},
			}
			uniqueKeys = append(uniqueKeys, newUK)
			currentUniqueKey = &uniqueKeys[len(uniqueKeys)-1]
		}

		// 同じ制約名の場合、現在のUniqueKeyのColumnsに追加
		currentUniqueKey.Columns = append(currentUniqueKey.Columns, columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return uniqueKeys, nil
}

// FetchForeignKeys テーブルの外部キー制約を取得
func (db *DB) FetchForeignKeys(ctx context.Context, dbName string, tableName string) ([]ForeignKey, error) {
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

	rows, err := db.conn.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// SQL取得の順序を維持しながら情報を構築する
	var foreignKeys []ForeignKey
	var currentFK *ForeignKey
	for rows.Next() {
		var constraintName, columnName, refTableName, refColumnName string
		if err := rows.Scan(&constraintName, &columnName, &refTableName, &refColumnName); err != nil {
			return nil, err
		}

		if currentFK == nil || currentFK.Name != constraintName {
			newFK := ForeignKey{
				Name:     constraintName,
				RefTable: refTableName,
			}
			foreignKeys = append(foreignKeys, newFK)
			currentFK = &foreignKeys[len(foreignKeys)-1]
		}

		currentFK.Columns = append(currentFK.Columns, columnName)
		currentFK.RefColumns = append(currentFK.RefColumns, refColumnName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return foreignKeys, nil
}

// FetchTableColumns テーブルのカラム情報を取得
func (db *DB) FetchTableColumns(ctx context.Context, dbName string, tableName string) ([]ColumnInfo, error) {
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

	rows, err := db.conn.QueryContext(ctx, query, dbName, tableName)
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

// FetchTableIndexes はテーブルのインデックス情報を取得するメソッド
func (db *DB) FetchTableIndexes(ctx context.Context, dbName string, tableName string) ([]IndexInfo, error) {
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

	rows, err := db.conn.QueryContext(ctx, query, dbName, tableName, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// SQL取得の順序を維持しながら情報を構築する
	var indexes []IndexInfo
	var currentIdx *IndexInfo
	for rows.Next() {
		var indexName, columnName string
		var nonUnique bool
		if err := rows.Scan(&indexName, &columnName, &nonUnique); err != nil {
			return nil, err
		}
		if currentIdx == nil || currentIdx.Name != indexName {
			newIdx := IndexInfo{
				Name:    indexName,
				Unique:  !nonUnique,
				Columns: []string{},
			}
			indexes = append(indexes, newIdx)
			currentIdx = &indexes[len(indexes)-1]
		}
		currentIdx.Columns = append(currentIdx.Columns, columnName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return indexes, nil
}
