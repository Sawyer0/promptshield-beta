-- Provider profiles (BYOK) table
-- Stores encrypted API keys per tenant for external LLM providers

CREATE TABLE IF NOT EXISTS provider_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  provider TEXT NOT NULL,           -- e.g., openai, anthropic, azure-openai
  label TEXT NOT NULL,              -- user-facing label
  api_key_encrypted TEXT NOT NULL,  -- AES-256-GCM ciphertext (base64)
  base_url TEXT,                    -- optional override
  extra_headers JSONB,              -- optional provider-specific headers
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Basic index for tenant scoping
CREATE INDEX IF NOT EXISTS idx_provider_profiles_tenant ON provider_profiles(tenant_id);

-- Row level security (optional): allow access only within tenant
-- Note: enable only if your deployment enforces current_setting('app.tenant_id')
DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'provider_profiles'
  ) THEN
    ALTER TABLE provider_profiles ENABLE ROW LEVEL SECURITY;
    CREATE POLICY provider_profiles_tenant_isolation ON provider_profiles
      USING (tenant_id::text = current_setting('app.tenant_id', true));
  END IF;
END $$;

