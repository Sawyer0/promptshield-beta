-- Check all tables in your Supabase database
-- Run this in your Supabase SQL Editor

-- Option 1: Simple list of all tables
SELECT 
    table_schema,
    table_name
FROM information_schema.tables
WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
ORDER BY table_schema, table_name;

-- Option 2: Tables with more details
SELECT 
    t.table_schema,
    t.table_name,
    t.table_type,
    obj.description as table_comment
FROM information_schema.tables t
LEFT JOIN pg_class c ON c.relname = t.table_name
LEFT JOIN pg_namespace n ON n.nspname = t.table_schema AND c.relnamespace = n.oid
LEFT JOIN pg_description obj ON obj.objoid = c.oid AND obj.objsubid = 0
WHERE t.table_schema NOT IN ('pg_catalog', 'information_schema')
ORDER BY t.table_schema, t.table_name;

-- Option 3: Count rows in each table (this will show which tables have data)
SELECT 
    schemaname,
    tablename,
    n_live_tup as row_count
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC;

-- Option 4: Just list table names in public schema (most common)
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER BY table_name;