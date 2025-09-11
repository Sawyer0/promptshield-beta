-- Check if your tables have data and the structure
-- Run these queries in your Supabase SQL Editor

-- 1. Check if default tenant exists
SELECT * FROM tenants 
WHERE id = '6f4d338d-f0c0-4091-b54e-f71752c8f568'
   OR name = 'default';

-- 2. Check all tenants
SELECT id, name, status FROM tenants;

-- 3. Check rulepacks
SELECT 
    r.id,
    r.name,
    r.tenant_id,
    r.status,
    t.name as tenant_name
FROM rulepacks r
LEFT JOIN tenants t ON r.tenant_id = t.id
LIMIT 10;

-- 4. Check table structure of rulepacks
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'rulepacks'
ORDER BY ordinal_position;

-- 5. Check table structure of tenants
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'tenants'
ORDER BY ordinal_position;