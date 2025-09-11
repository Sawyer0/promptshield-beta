-- Ensure tenants has deleted_at column for backend compatibility
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

