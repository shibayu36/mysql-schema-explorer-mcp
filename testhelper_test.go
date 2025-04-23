package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// setupTestDB はテスト用のMySQLコンテナを起動し、DB接続とクリーンアップ関数を返します。
// スキーマの適用は別途 applySchema で行います。
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Dockerプールの初期化
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not construct pool: %s", err)
	}

	// Dockerホストに接続できるか確認
	err = pool.Client.Ping()
	if err != nil {
		t.Fatalf("Could not connect to Docker: %s", err)
	}

	// MySQLコンテナの起動設定
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mysql",
		Tag:        "8.0", // 任意のMySQLバージョンを指定
		Env: []string{
			"MYSQL_ROOT_PASSWORD=secret",
			"MYSQL_DATABASE=test_db",
		},
	}, func(config *docker.HostConfig) {
		// Set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("Could not start resource: %s", err)
	}

	// コンテナのポートを取得
	mysqlPort := resource.GetPort("3306/tcp")
	dsn := fmt.Sprintf("root:secret@(localhost:%s)/test_db?parseTime=true", mysqlPort)

	// クリーンアップ関数: コンテナの停止と削除
	cleanup := func() {
		if err := pool.Purge(resource); err != nil {
			t.Fatalf("Could not purge resource: %s", err)
		}
	}

	var db *sql.DB

	// データベース接続のリトライ（コンテナ起動直後は接続できない場合があるため）
	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		cleanup() // エラー時はクリーンアップを実行
		t.Fatalf("Could not connect to database: %s", err)
	}

	t.Logf("Test database container is ready on port %s", mysqlPort)

	// 接続とクリーンアップ関数を返す
	// cleanup関数をテスト関数の最後にdeferで呼び出す必要がある
	return db, cleanup
}

// applySchema は指定されたスキーマファイルをDBに適用します。
func applySchema(t *testing.T, db *sql.DB, schemaPath string) {
	t.Helper()

	// スキーマファイルの読み込み
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Could not read schema file %s: %s", schemaPath, err)
	}
	schema := string(schemaBytes)

	// SQL文を分割（単純な";"区切りで分割）
	statements := strings.Split(schema, ";")

	// 各SQL文を実行
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err := db.Exec(stmt)
		if err != nil {
			log.Printf("Failed to execute statement from %s: %s\nError: %v", schemaPath, stmt, err) // 実行に失敗したSQLをログ出力
			t.Fatalf("Could not apply schema from %s: %v", schemaPath, err)
		}
	}
	t.Logf("Applied schema from %s", schemaPath)
}
