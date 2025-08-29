package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Example showing how to integrate tenant-aware database operations
func main() {
	// This example shows how your application should handle tenant isolation

	dsn := os.Getenv("PS_PG_DSN")
	if dsn == "" {
		log.Fatal("PS_PG_DSN environment variable is required")
	}

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Create router with tenant middleware
	r := chi.NewRouter()
	
	// Add tenant validation middleware (this would come from your main router)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract tenant ID from header
			tenantIDStr := r.Header.Get("X-PS-Tenant-ID")
			if tenantIDStr == "" {
				http.Error(w, "Missing X-PS-Tenant-ID header", http.StatusBadRequest)
				return
			}

			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
				return
			}

			// Set tenant context in the request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "db.tenant_id", tenantID.String())

			// CRITICAL: Set tenant context in database for RLS
			if err := setTenantContext(db, tenantID); err != nil {
				log.Printf("Failed to set tenant context: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// Example endpoint: List rulepacks (tenant-isolated)
	r.Get("/rulepacks", func(w http.ResponseWriter, r *http.Request) {
		// Thanks to RLS, this query will automatically only return
		// rulepacks for the current tenant
		rulepacks, err := getRulePacksForTenant(r.Context(), db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"rulepacks": %d, "message": "RLS automatically filtered to current tenant"}`, len(rulepacks))
	})

	// Example endpoint: Create rulepack (tenant-isolated)  
	r.Post("/rulepacks", func(w http.ResponseWriter, r *http.Request) {
		// The INSERT will automatically include the tenant_id
		// from the current tenant context set by RLS
		rulepackID, err := createRulePackForTenant(r.Context(), db, "New Security Rules")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id": "%s", "message": "Created rulepack for current tenant"}`, rulepackID)
	})

	// Example endpoint: Test tenant isolation
	r.Get("/test-isolation", func(w http.ResponseWriter, r *http.Request) {
		result := testTenantIsolation(db)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"isolation_test": "%s"}`, result)
	})

	fmt.Println("🚀 Starting tenant isolation example server on :8080")
	fmt.Println("📝 Test endpoints:")
	fmt.Println("  GET  /rulepacks - List tenant rulepacks")
	fmt.Println("  POST /rulepacks - Create tenant rulepack") 
	fmt.Println("  GET  /test-isolation - Test isolation")
	fmt.Println()
	fmt.Println("🧪 Example requests:")
	fmt.Println(`  curl -H "X-PS-Tenant-ID: 11111111-1111-1111-1111-111111111111" http://localhost:8080/rulepacks`)
	fmt.Println(`  curl -H "X-PS-Tenant-ID: 22222222-2222-2222-2222-222222222222" http://localhost:8080/rulepacks`)

	log.Fatal(http.ListenAndServe(":8080", r))
}

// setTenantContext calls the database function to set tenant context for RLS
func setTenantContext(db *sql.DB, tenantID uuid.UUID) error {
	_, err := db.Exec("SELECT set_tenant_context($1::uuid)", tenantID)
	return err
}

// getRulePacksForTenant retrieves rulepacks - RLS automatically filters by tenant
func getRulePacksForTenant(ctx context.Context, db *sql.DB) ([]map[string]interface{}, error) {
	// Thanks to RLS, this query automatically filters by current tenant
	query := `
		SELECT id, name, description, enabled, created_at
		FROM rulepacks 
		WHERE enabled = true
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rulepacks: %w", err)
	}
	defer rows.Close()

	var rulepacks []map[string]interface{}
	for rows.Next() {
		var id, name, description string
		var enabled bool
		var createdAt string

		if err := rows.Scan(&id, &name, &description, &enabled, &createdAt); err != nil {
			continue
		}

		rulepacks = append(rulepacks, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
			"enabled":     enabled,
			"created_at":  createdAt,
		})
	}

	return rulepacks, nil
}

// createRulePackForTenant creates a rulepack - RLS automatically sets tenant_id
func createRulePackForTenant(ctx context.Context, db *sql.DB, name string) (uuid.UUID, error) {
	rulepackID := uuid.New()

	// Note: We don't need to explicitly set tenant_id in the INSERT
	// The RLS policies will automatically ensure it's set correctly
	query := `
		INSERT INTO rulepacks (id, tenant_id, name, description, enabled, metadata, rules)
		VALUES ($1, (SELECT get_current_tenant_id()), $2, $3, true, '{"version": "1.0.0"}', '[]')
		RETURNING id
	`

	var returnedID uuid.UUID
	err := db.QueryRowContext(ctx, query, rulepackID, name, "Auto-generated rulepack").Scan(&returnedID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to create rulepack: %w", err)
	}

	return returnedID, nil
}

// testTenantIsolation demonstrates that different tenants see different data
func testTenantIsolation(db *sql.DB) string {
	// Test tenant 1
	tenant1, _ := uuid.Parse("11111111-1111-1111-1111-111111111111")
	setTenantContext(db, tenant1)

	var count1 int
	db.QueryRow("SELECT COUNT(*) FROM rulepacks").Scan(&count1)

	// Test tenant 2  
	tenant2, _ := uuid.Parse("22222222-2222-2222-2222-222222222222")
	setTenantContext(db, tenant2)

	var count2 int
	db.QueryRow("SELECT COUNT(*) FROM rulepacks").Scan(&count2)

	return fmt.Sprintf("Tenant1: %d rulepacks, Tenant2: %d rulepacks - Isolation: %s", 
		count1, count2, map[bool]string{true: "SUCCESS", false: "FAILED"}[count1 != count2])
}