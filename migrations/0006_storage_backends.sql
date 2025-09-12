-- 0006_storage_backends.sql
-- Storage backend catalog for object storage (S3/GCS) configuration

CREATE TABLE storage_backends (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  provider TEXT NOT NULL CHECK (provider IN ('s3','gcs')),
  bucket TEXT NOT NULL,
  region TEXT,
  endpoint TEXT,
  prefix TEXT NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Exactly one active backend
CREATE UNIQUE INDEX storage_backends_one_active ON storage_backends (is_active) WHERE is_active = true;

CREATE OR REPLACE FUNCTION set_active_storage_backend(p_id UUID)
RETURNS VOID AS $$
BEGIN
  UPDATE storage_backends SET is_active=false;
  UPDATE storage_backends SET is_active=true, updated_at=now() WHERE id = p_id;
END; $$ LANGUAGE plpgsql;

