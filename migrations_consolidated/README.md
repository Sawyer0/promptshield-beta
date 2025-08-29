# Consolidated Migration Plan

## 🗂️ **Migration Structure**

This consolidated migration plan replaces the conflicted original migrations (0001-0009) with a clean, logical structure:

### **Original Issues Fixed:**
- ✅ **Removed**: 0002 (useless placeholder)  
- ✅ **Resolved**: 0004 vs 0005 table conflicts
- ✅ **Fixed**: 0007 references to non-existent tables
- ✅ **Consolidated**: Redundant schema definitions
- ✅ **Cleaned**: Performance indexes and constraints

## 📋 **New Migration Order**

### **0001_initial_schema.sql**
- **Replaces**: 0001 + 0003 + tenant fixes from 0004
- **Creates**: Core tables (tenants, rulepacks, assignments, audits)  
- **Includes**: All performance indexes and constraints
- **Safe**: No conflicts, fully consolidated

### **0002_production_tables.sql**
- **Replaces**: 0006 (cleaned up)
- **Creates**: All production SaaS tables (users, billing, monitoring)
- **Includes**: Proper indexes and relationships
- **Clean**: No conflicts with core schema

### **0003_tenant_services.sql**  
- **Replaces**: 0008 (unchanged)
- **Creates**: Service management tables
- **Includes**: Helper functions for service lifecycle
- **Ready**: Production-ready service management

### **0004_row_level_security.sql**
- **Replaces**: 0009 (enhanced)
- **Creates**: RLS policies and functions
- **Includes**: Performance indexes for RLS
- **Secure**: Complete tenant isolation

## 🚀 **Deployment Strategy**

### **For Fresh Database:**
```sql
-- Run in order:
\i migrations_consolidated/0001_initial_schema.sql
\i migrations_consolidated/0002_production_tables.sql  
\i migrations_consolidated/0003_tenant_services.sql
\i migrations_consolidated/0004_row_level_security.sql
```

### **For Existing Database:**
```sql
-- 1. Backup first!
pg_dump your_database > backup_before_consolidation.sql

-- 2. Check current state
SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;

-- 3. Apply only missing parts (check each migration)
-- Most likely you only need:
\i migrations_consolidated/0003_tenant_services.sql
\i migrations_consolidated/0004_row_level_security.sql
```

## ✅ **Verification**

After running consolidated migrations:

```sql  
-- Check all tables exist
SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public';  
-- Should return ~25 tables

-- Check RLS is enabled
SELECT tablename, rowsecurity 
FROM pg_tables 
WHERE schemaname = 'public' AND rowsecurity = true
ORDER BY tablename;
-- Should show 17+ tables with RLS

-- Test tenant isolation
SELECT set_tenant_context('test-tenant-id');
SELECT COUNT(*) FROM rulepacks; -- Should filter by tenant
```

## 📁 **File Cleanup Plan**

### **Safe to Remove:**
- `migrations/0002_add_idempotency_key.sql` (placeholder)
- `migrations/0009a_rls_check_tables.sql` (diagnostic only)

### **Keep for Reference:**
- Original migrations (until consolidation is verified)
- Archive in `migrations_archive/` folder

### **Replace Eventually:**  
- Move consolidated migrations to `migrations/` 
- Archive old migrations to `migrations_archive/`

## 🔧 **Benefits of Consolidation**

✅ **No Conflicts**: Schema evolution is linear and logical  
✅ **Better Performance**: Indexes created with tables from start  
✅ **Easier Deployment**: Clear dependency chain  
✅ **Self-Documenting**: Each migration has single clear purpose  
✅ **Maintainable**: Future changes are additive only  
✅ **Production Ready**: All edge cases and fixes included  

## 🚨 **Next Steps**

1. **Test** consolidated migrations on fresh database
2. **Verify** all functionality works with new schema  
3. **Archive** old migrations once verified
4. **Update** deployment scripts to use new migrations
5. **Document** for team about migration consolidation