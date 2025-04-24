package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// createTestDBConfig テスト用のDB設定を作成。環境変数がなければデフォルト値を使用
func createTestDBConfig(t *testing.T) DBConfig {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "13306"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "root"
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "rootpass"
	}

	return DBConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}
}

const testDBName = "test_mysql_schema_explorer_mcp"

// setupTestDB はテスト用DBを作成し、接続を返します。
// テスト終了後にクリーンアップ関数を呼び出すことでDBを削除します。
func setupTestDB(t *testing.T, schemaFile string) *sql.DB {
	t.Helper()

	config := createTestDBConfig(t)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/",
		config.User, config.Password, config.Host, config.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("MySQL接続に失敗: %v", err)
	}

	// DBの作成（既存のDBがあれば削除）
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName))
	if err != nil {
		db.Close()
		t.Fatalf("データベース削除に失敗: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE `%s`", testDBName))
	if err != nil {
		db.Close()
		t.Fatalf("データベース作成に失敗: %v", err)
	}

	// スキーマの適用
	{
		applyDB, err := sql.Open("mysql", dsn+testDBName)
		if err != nil {
			t.Fatalf("テスト用DBへの接続に失敗: %v", err)
		}
		defer applyDB.Close()

		schemaBytes, err := os.ReadFile(schemaFile)
		if err != nil {
			t.Fatalf("スキーマファイル読み込みに失敗: %v", err)
		}
		schema := string(schemaBytes)

		// SQL文を分割して実行
		statements := strings.Split(schema, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			_, err := applyDB.Exec(stmt)
			if err != nil {
				t.Logf("実行に失敗したSQL: %s", stmt)
				t.Fatalf("スキーマ適用に失敗: %v", err)
			}
		}
	}

	t.Cleanup(func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName))
		db.Close()
	})

	return db
}
