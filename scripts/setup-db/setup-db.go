package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	
	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from command line or environment
	dbURL := "postgresql://postgres:hVygdTBDDT4FfoXp@db.jsrlqqtfhjfiawxkmoqe.supabase.co:5432/postgres"
	if len(os.Args) > 1 {
		dbURL = os.Args[1]
	}
	
	fmt.Println("🚀 PromptShield Database Setup")
	fmt.Println("==============================")
	fmt.Printf("Connecting to database...\n")
	
	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()
	
	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	fmt.Println("✅ Connected successfully!")
	
	// Read and execute migrations
	migrations := []string{
		"migrations/0001_init.sql",
		"migrations/0002_add_idempotency_key.sql", 
		"migrations/0003_schema_fixes.sql",
		"migrations/0004_enterprise_schema.sql",
		"migrations/0005_unified_schema.sql",
		"migrations/0006_complete_production_schema.sql",
		"migrations/0007_cleanup_redundant_tables.sql",
		"migrations/0008_tenant_services.sql",
		"migrations/0009_rls_policies.sql",
	}
	
	for _, migration := range migrations {
		content, err := os.ReadFile(migration)
		if err != nil {
			fmt.Printf("⚠️  Skipping %s: %v\n", migration, err)
			continue
		}
		
		fmt.Printf("Running %s...\n", migration)
		if _, err := db.Exec(string(content)); err != nil {
			fmt.Printf("⚠️  Migration %s failed (may already exist): %v\n", migration, err)
		} else {
			fmt.Printf("✅ %s completed\n", migration)
		}
	}
	
	// Create default tenant
	fmt.Println("\n👤 Creating default tenant...")
	var tenantID string
	err = db.QueryRow(`
		INSERT INTO tenants (id, name) 
		VALUES (gen_random_uuid(), 'default-tenant') 
		ON CONFLICT (name) DO UPDATE SET name = 'default-tenant'
		RETURNING id
	`).Scan(&tenantID)
	
	if err != nil {
		// Try to get existing tenant
		err = db.QueryRow(`SELECT id FROM tenants WHERE name = 'default-tenant'`).Scan(&tenantID)
		if err != nil {
			log.Fatalf("Failed to create/get tenant: %v", err)
		}
	}
	
	fmt.Printf("✅ Tenant ID: %s\n", tenantID)
	
	// Print environment setup
	fmt.Println("\n📝 Add these to your .env file or export them:")
	fmt.Println("================================================")
	fmt.Printf("export PS_PG_DSN=\"%s\"\n", dbURL)
	fmt.Printf("export PS_TENANT_ID=\"%s\"\n", tenantID)
	fmt.Println("export PS_CONTROL_PLANE_ADDR=\":8085\"")
	fmt.Println("export PS_ENFORCER_ADDR=\":9090\"")
	fmt.Println("export PS_ENFORCER_GRPC_ADDR=\":9091\"")
	
	fmt.Println("\n🎉 Setup complete! You can now run:")
	fmt.Println("  make build")
	fmt.Println("  ./bin/ps-gateway")
}