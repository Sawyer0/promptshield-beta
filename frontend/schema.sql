-- Schema for PromptShield Frontend Database
-- This creates all necessary tables for your application

-- Users table (for Replit Auth)
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT,
    first_name TEXT,
    last_name TEXT,
    profile_image_url TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Policies table
CREATE TABLE IF NOT EXISTS policies (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('security', 'compliance', 'custom')),
    content TEXT NOT NULL,
    version INTEGER DEFAULT 1,
    created_by TEXT REFERENCES users(id),
    is_active BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Violations table
CREATE TABLE IF NOT EXISTS violations (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    request_id TEXT NOT NULL,
    policy_id TEXT,
    content TEXT,
    decision TEXT NOT NULL,
    reason TEXT,
    severity TEXT,
    rule_matched TEXT,
    processing_time_ms INTEGER,
    metadata JSONB,
    timestamp TIMESTAMP DEFAULT NOW()
);

-- RulePacks table
CREATE TABLE IF NOT EXISTS rulepacks (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name TEXT NOT NULL,
    description TEXT,
    version TEXT,
    author TEXT,
    tags TEXT[],
    category TEXT,
    rules_count INTEGER,
    status TEXT DEFAULT 'draft',
    content JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- System metrics table
CREATE TABLE IF NOT EXISTS system_metrics (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    metric_name TEXT NOT NULL,
    metric_value FLOAT,
    metric_type TEXT,
    timestamp TIMESTAMP DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_policies_active ON policies(is_active);
CREATE INDEX IF NOT EXISTS idx_violations_timestamp ON violations(timestamp);
CREATE INDEX IF NOT EXISTS idx_violations_severity ON violations(severity);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);