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
var migrationFile string
if len(os.Args) >= 2 {
	migrationFile = os.Args[1]
}

dsn := os.Getenv("PS_PG_DSN")
	
	if dsn == "" {
		dsn = "postgresql://postgres:hVygdTBDDT4FfoXp@db.jsrlqqtfhjfiawxkmoqe.supabase.co:5432/postgres"
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

// Execute migration(s)
if migrationFile != "" {
	// Single file mode
	fmt.Printf("📁 Running migration: %s\n", migrationFile)
	content, err := ioutil.ReadFile(migrationFile)
	if err != nil { log.Fatalf("Failed to read migration file: %v", err) }
	if _, err := db.Exec(string(content)); err != nil { log.Fatalf("Failed to execute migration: %v", err) }
	fmt.Printf("✅ Migration completed successfully!\n")
} else {
	// Run all *.sql in migrations directory, sorted
	dir := "migrations"
	entries, err := ioutil.ReadDir(dir)
	if err != nil { log.Fatalf("Failed to read migrations dir: %v", err) }
	for _, e := range entries {
		if e.IsDir() { continue }
		name := e.Name()
		if len(name) < 5 || name[len(name)-4:] != ".sql" { continue }
		path := dir + string(os.PathSeparator) + name
		fmt.Printf("📁 Running migration: %s\n", path)
		content, err := ioutil.ReadFile(path)
		if err != nil { log.Fatalf("Failed to read migration %s: %v", path, err) }
		if _, err := db.Exec(string(content)); err != nil { log.Fatalf("Failed to execute migration %s: %v", path, err) }
	}
	fmt.Printf("✅ All migrations completed successfully!\n")
}

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