package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run run-migration.go <migration-file>")
	}

	migrationFile := os.Args[1]
	dsn := os.Getenv("PS_PG_DSN")
	
	if dsn == "" {
		dsn = "postgresql://postgres:hVygdTBDDT4FfoXp@db.jsrlqqtfhjfiawxkmoqe.supabase.co:5432/postgres"
	}

	// Read migration file
	content, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Printf("🔌 Connected to database\n")
	fmt.Printf("📁 Running migration: %s\n", migrationFile)

	// Execute migration
	_, err = db.Exec(string(content))
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	fmt.Printf("✅ Migration completed successfully!\n")

	// Verify RLS is enabled
	fmt.Printf("🔍 Verifying RLS policies...\n")
	
	rows, err := db.Query(`
		SELECT schemaname, tablename, rowsecurity 
		FROM pg_tables 
		WHERE schemaname = 'public' AND rowsecurity = true
		ORDER BY tablename;
	`)
	if err != nil {
		log.Printf("Warning: Failed to verify RLS: %v", err)
		return
	}
	defer rows.Close()

	fmt.Printf("📋 Tables with RLS enabled:\n")
	count := 0
	for rows.Next() {
		var schema, table string
		var rls bool
		if err := rows.Scan(&schema, &table, &rls); err != nil {
			continue
		}
		fmt.Printf("  ✓ %s.%s\n", schema, table)
		count++
	}
	
	fmt.Printf("🎯 Total tables with RLS: %d\n", count)
}