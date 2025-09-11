package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if strings.TrimSpace(migrationsDir) == "" {
		migrationsDir = "migrations_aurora"
	}

	dsn := os.Getenv("AURORA_PG_DSN")
	if strings.TrimSpace(dsn) == "" {
		log.Fatal("AURORA_PG_DSN is required (e.g. postgres://user:pass@cluster-writer:5432/db?sslmode=require)")
	}

	files, err := listSQLFiles(migrationsDir)
	if err != nil {
		log.Fatalf("failed to list migrations: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("no .sql files found in %s", migrationsDir)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("db ping failed: %v", err)
	}

	fmt.Printf("Applying %d migrations from %s\n", len(files), migrationsDir)
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("read %s: %v", f, err)
		}
		fmt.Printf("-> %s... ", filepath.Base(f))
		if _, err := db.Exec(string(content)); err != nil {
			fmt.Println("FAIL")
			log.Fatalf("execute %s failed: %v", f, err)
		}
		fmt.Println("OK")
	}

	fmt.Println("All migrations applied successfully.")
}

func listSQLFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }
		if d.IsDir() { return nil }
		if strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil { return nil, err }
	sort.Strings(files)
	return files, nil
}

