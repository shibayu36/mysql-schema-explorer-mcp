package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// createTestDBConfig creates DB settings for testing. Uses default values if environment variables are not set.
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

// setupTestDB creates a test DB and returns the connection.
// It deletes the DB by calling the cleanup function after the test finishes.
func setupTestDB(t *testing.T, schemaFile string) *sql.DB {
	t.Helper()

	config := createTestDBConfig(t)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/",
		config.User, config.Password, config.Host, config.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// Create DB (delete if exists)
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName))
	if err != nil {
		db.Close()
		t.Fatalf("Failed to drop database: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE `%s`", testDBName))
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create database: %v", err)
	}

	// Apply schema
	{
		applyDB, err := sql.Open("mysql", dsn+testDBName)
		if err != nil {
			t.Fatalf("Failed to connect to test DB: %v", err)
		}
		defer applyDB.Close()

		schemaBytes, err := os.ReadFile(schemaFile)
		if err != nil {
			t.Fatalf("Failed to read schema file: %v", err)
		}
		schema := string(schemaBytes)

		// Split and execute SQL statements
		statements := strings.Split(schema, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			_, err := applyDB.Exec(stmt)
			if err != nil {
				t.Logf("Failed to execute SQL: %s", stmt)
				t.Fatalf("Failed to apply schema: %v", err)
			}
		}
	}

	t.Cleanup(func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName))
		db.Close()
	})

	return db
}
