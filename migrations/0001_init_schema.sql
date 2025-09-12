-- 0001_init_schema.sql
-- Clean end-to-end database architecture aligned with DATABASE_ARCHITECTURE.md

-- ============ Extensions ============
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- Optional: BRIN benefits for append-only time partitions
CREATE EXTENSION IF NOT EXISTS btree_gin;
-- Optional, if available in your Aurora/Postgres: scheduling
DO $$ BEGIN EXECUTE 'CREATE EXTENSION IF NOT EXISTS pg_cron'; EXCEPTION WHEN others THEN NULL; END $$;

-- ============ Tenant Context & Admin ============
CREATE OR REPLACE FUNCTION set_tenant_context(tenant_id UUID)
RETURNS VOID AS $$
BEGIN
  PERFORM set_config('app.current_tenant_id', tenant_id::text, true);
END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS UUID AS $$
BEGIN
  RETURN COALESCE(current_setting('app.current_tenant_id', true)::UUID,
                  '00000000-0000-0000-0000-000000000000'::UUID);
END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION set_platform_admin(is_admin BOOL)
RETURNS VOID AS $$
BEGIN
  PERFORM set_config('app.platform_admin', CASE WHEN is_admin THEN 'true' ELSE 'false' END, true);
END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION is_platform_admin()
RETURNS BOOL AS $$
BEGIN
  RETURN COALESCE(current_setting('app.platform_admin', true), 'false')::BOOL;
END; $$ LANGUAGE plpgsql;

-- ============ Core Entities ============
CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'active',          -- active|suspended|deleted
  plan_type TEXT NOT NULL DEFAULT 'free',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE platform_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  display_name TEXT,
  roles TEXT[] NOT NULL DEFAULT ARRAY['platform_admin'],
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  display_name TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(email)
);

CREATE TABLE tenant_memberships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member',            -- admin|developer|auditor|member
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, user_id)
);

CREATE TABLE platform_settings (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ RulePacks & Versions ============
CREATE TABLE rulepacks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  yaml_content TEXT,                   -- raw YAML
  rules JSONB,                         -- parsed/compiled rules
  current_version_id UUID,
  is_active BOOLEAN NOT NULL DEFAULT true,
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('draft','active','archived')),
  enforcement_mode TEXT NOT NULL DEFAULT 'enforce' CHECK (enforcement_mode IN ('monitor','enforce','redact')),
  fail_on_severity TEXT NOT NULL DEFAULT 'HIGH' CHECK (fail_on_severity IN ('LOW','MEDIUM','HIGH','CRITICAL')),
  priority INT NOT NULL DEFAULT 100,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  rule_count INT NOT NULL DEFAULT 0,
  content_size_bytes INT NOT NULL DEFAULT 0,
  last_compiled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rulepack_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
  version INT NOT NULL,
  yaml_content TEXT NOT NULL,
  dsl JSONB NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','approved','active','archived')),
  created_by UUID,
  approval_workflow JSONB,
  rule_count INT NOT NULL DEFAULT 0,
  content_size_bytes INT NOT NULL DEFAULT 0,
  compilation_time_ms INT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(rulepack_id, version)
);

-- ============ Assignments ============
CREATE TABLE rulepack_assignments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
  target_scope TEXT NOT NULL,          -- e.g., /v1/tools/:id or wildcard paths
  method TEXT NOT NULL DEFAULT 'ANY',  -- e.g., GET|POST|ANY
  priority INT NOT NULL DEFAULT 100,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  first_path_segment TEXT GENERATED ALWAYS AS (split_part(target_scope,'/',2)) STORED
);

-- ============ Tools & Tenant Settings ============
CREATE TABLE tools (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  tool_id TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT DEFAULT '',
  capability_tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  data_domains JSONB NOT NULL DEFAULT '[]'::jsonb,
  side_effect TEXT NOT NULL DEFAULT 'none',
  auth_scope TEXT NOT NULL DEFAULT 'user-delegated',
  arg_schema JSONB,
  risk_score INT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, tool_id)
);

CREATE TABLE tenant_settings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  value JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, key)
);

-- ============ Real-time Operations (Partitioned) ============
CREATE TABLE violations (
  id UUID DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
  rule_id TEXT NOT NULL,
  severity TEXT NOT NULL CHECK (severity IN ('LOW','MEDIUM','HIGH','CRITICAL')),
  message TEXT NOT NULL,
  input_hash TEXT NOT NULL,
  scan_result JSONB,
  enforcement_action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE scan_results (
  id UUID DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  request_id TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  method TEXT NOT NULL,
  scan_status TEXT NOT NULL,
  violations_count INT NOT NULL DEFAULT 0,
  processing_time_ms INT,
  decision JSONB,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE audits (
  id UUID DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  actor_id UUID,
  actor_email TEXT,
  action TEXT NOT NULL,
  object_type TEXT NOT NULL,
  object_id UUID NOT NULL,
  diff JSONB,
  metadata JSONB,
  integrity_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create current and next month partitions
DO $$
DECLARE
  bases TEXT[] := ARRAY['violations','scan_results','audits'];
  base TEXT; s1 DATE := date_trunc('month', now())::date; s2 DATE := (date_trunc('month', now()) + INTERVAL '1 month')::date;
  e1 DATE := (s1 + INTERVAL '1 month')::date; e2 DATE := (s2 + INTERVAL '1 month')::date;
  p1 TEXT; p2 TEXT;
BEGIN
  FOREACH base IN ARRAY bases LOOP
    p1 := format('%s_%s', base, to_char(s1,'YYYY_MM'));
    p2 := format('%s_%s', base, to_char(s2,'YYYY_MM'));
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L);', p1, base, s1, e1);
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L);', p2, base, s2, e2);
  END LOOP;
END $$;

-- ============ Billing & Subscriptions ============
CREATE TABLE subscription_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  description TEXT,
  price_monthly INTEGER NOT NULL, -- in cents
  price_yearly INTEGER, -- in cents
  features JSONB NOT NULL DEFAULT '{}'::jsonb,
  limits JSONB NOT NULL DEFAULT '{}'::jsonb, -- {"api_calls_monthly": 1000000, "llm_calls_monthly": 10000, "rulepacks": -1}
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  plan_id UUID NOT NULL REFERENCES subscription_plans(id),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('trial', 'active', 'past_due', 'canceled', 'suspended')),
  billing_cycle TEXT DEFAULT 'monthly' CHECK (billing_cycle IN ('monthly', 'yearly')),
  stripe_customer_id TEXT,
  stripe_subscription_id TEXT,
  current_period_start TIMESTAMPTZ,
  current_period_end TIMESTAMPTZ,
  trial_ends_at TIMESTAMPTZ,
  canceled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- LLM usage tracking for billing
CREATE TABLE llm_usage (
  id UUID DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  usage_date DATE NOT NULL,
  llm_calls_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (id, usage_date),
  UNIQUE(tenant_id, usage_date)
) PARTITION BY RANGE (usage_date);

-- Usage-based billing records
CREATE TABLE usage_billing (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  subscription_id UUID NOT NULL REFERENCES subscriptions(id),
  billing_period_start DATE NOT NULL,
  billing_period_end DATE NOT NULL,
  included_llm_calls INTEGER NOT NULL,
  used_llm_calls INTEGER NOT NULL,
  overage_llm_calls INTEGER NOT NULL,
  overage_cost_cents INTEGER NOT NULL,
  total_cost_cents INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'billed', 'paid', 'failed')),
  stripe_invoice_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Insert default subscription plans
INSERT INTO subscription_plans (name, display_name, price_monthly, price_yearly, features, limits) VALUES
  ('professional', 'Professional', 29900, 299000, 
   '["Unlimited rulepacks", "1M API calls/month", "10K LLM calls/month", "SOC2/GDPR compliance", "Email support", "99.5% SLA"]',
   '{"api_calls_monthly": 1000000, "llm_calls_monthly": 10000, "rulepacks": -1, "rules_per_pack": -1, "compliance_frameworks": 2}'),
   
  ('enterprise', 'Enterprise', 199900, 1999000,
   '["Unlimited API calls", "100K LLM calls/month", "Advanced compliance", "24/7 support", "Custom integrations", "99.9% SLA"]',
   '{"api_calls_monthly": -1, "llm_calls_monthly": 100000, "rulepacks": -1, "rules_per_pack": -1, "compliance_frameworks": -1}'),
   
  ('enterprise_plus', 'Enterprise Plus', 0, 0, -- Custom pricing
   '["Unlimited everything", "1M+ LLM calls/month", "Custom frameworks", "On-premise options", "White-labeling", "Priority development"]',
   '{"api_calls_monthly": -1, "llm_calls_monthly": 1000000, "rulepacks": -1, "rules_per_pack": -1, "compliance_frameworks": -1}');

-- Invoice generation and management
CREATE TABLE invoices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  subscription_id UUID NOT NULL REFERENCES subscriptions(id),
  invoice_number TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'sent', 'paid', 'overdue', 'void')),
  billing_period_start DATE NOT NULL,
  billing_period_end DATE NOT NULL,
  subtotal_cents INTEGER NOT NULL,
  tax_cents INTEGER NOT NULL DEFAULT 0,
  total_cents INTEGER NOT NULL,
  currency TEXT NOT NULL DEFAULT 'usd',
  due_date DATE NOT NULL,
  paid_at TIMESTAMPTZ,
  stripe_invoice_id TEXT,
  pdf_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE invoice_line_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
  description TEXT NOT NULL,
  quantity INTEGER NOT NULL,
  unit_price_cents INTEGER NOT NULL,
  total_cents INTEGER NOT NULL,
  line_type TEXT NOT NULL CHECK (line_type IN ('subscription', 'usage', 'overage', 'discount', 'tax')),
  metadata JSONB DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Invoice templates for different plan types
CREATE TABLE invoice_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  plan_name TEXT NOT NULL,
  template_name TEXT NOT NULL,
  subject_template TEXT NOT NULL,
  body_template TEXT NOT NULL,
  footer_template TEXT,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(plan_name, template_name)
);

-- Insert default invoice templates
INSERT INTO invoice_templates (plan_name, template_name, subject_template, body_template, footer_template) VALUES
  ('professional', 'monthly', 'Your PromptShield Professional Invoice - {{invoice_number}}', 
   'Thank you for using PromptShield Professional. Your monthly subscription and usage charges are detailed below.', 
   'Questions? Contact us at billing@promptshield.com'),
  ('enterprise', 'monthly', 'Your PromptShield Enterprise Invoice - {{invoice_number}}', 
   'Thank you for using PromptShield Enterprise. Your monthly subscription and usage charges are detailed below.', 
   'Questions? Contact us at enterprise@promptshield.com'),
  ('enterprise_plus', 'monthly', 'Your PromptShield Enterprise Plus Invoice - {{invoice_number}}', 
   'Thank you for using PromptShield Enterprise Plus. Your monthly subscription and usage charges are detailed below.', 
   'Questions? Contact us at enterprise@promptshield.com');

-- A/B Testing and Experiment Management
CREATE TABLE experiments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  description TEXT,
  type TEXT NOT NULL CHECK (type IN ('pricing_tier', 'usage_addon', 'feature_flag', 'ui_optimization')),
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'paused', 'completed', 'cancelled')),
  traffic_allocation DECIMAL(3,2) NOT NULL DEFAULT 1.00 CHECK (traffic_allocation >= 0.00 AND traffic_allocation <= 1.00),
  target_criteria JSONB DEFAULT '{}'::jsonb,
  start_date TIMESTAMPTZ,
  end_date TIMESTAMPTZ,
  primary_metric TEXT NOT NULL,
  secondary_metrics TEXT[] DEFAULT '{}',
  results JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE experiment_variants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  experiment_id UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('control', 'test')),
  weight DECIMAL(3,2) NOT NULL DEFAULT 0.50 CHECK (weight >= 0.00 AND weight <= 1.00),
  configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
  results JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE experiment_assignments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  experiment_id UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
  variant_id UUID NOT NULL REFERENCES experiment_variants(id) ON DELETE CASCADE,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  first_seen_at TIMESTAMPTZ,
  last_seen_at TIMESTAMPTZ,
  converted_at TIMESTAMPTZ,
  UNIQUE(user_id, tenant_id, experiment_id)
);

CREATE TABLE experiment_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  assignment_id UUID NOT NULL REFERENCES experiment_assignments(id) ON DELETE CASCADE,
  event_name TEXT NOT NULL,
  event_data JSONB DEFAULT '{}'::jsonb,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create LLM usage partitions for current and next month
DO $$
DECLARE
  s1 DATE := date_trunc('month', now())::date;
  s2 DATE := (date_trunc('month', now()) + INTERVAL '1 month')::date;
  e1 DATE := (s1 + INTERVAL '1 month')::date;
  e2 DATE := (s2 + INTERVAL '1 month')::date;
  p1 TEXT; p2 TEXT;
BEGIN
  p1 := format('llm_usage_%s', to_char(s1,'YYYY_MM'));
  p2 := format('llm_usage_%s', to_char(s2,'YYYY_MM'));
  EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF llm_usage FOR VALUES FROM (%L) TO (%L);', p1, s1, e1);
  EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF llm_usage FOR VALUES FROM (%L) TO (%L);', p2, s2, e2);
END $$;

-- ============ Analytics & Compliance ============
CREATE TABLE usage_metrics (
  id UUID DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  metric_type TEXT NOT NULL,
  metric_value NUMERIC NOT NULL,
  dimensions JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE performance_metrics (
  id UUID DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  operation TEXT NOT NULL,
  latency_ms INT NOT NULL,
  throughput_rps NUMERIC,
  error_rate NUMERIC,
  resource_usage JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE compliance_evidence (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  standard TEXT NOT NULL,
  requirement_id TEXT NOT NULL,
  evidence_type TEXT NOT NULL,
  time_range_start TIMESTAMPTZ NOT NULL,
  time_range_end TIMESTAMPTZ NOT NULL,
  event_count INT NOT NULL,
  evidence_data JSONB NOT NULL,
  integrity_hash TEXT NOT NULL,
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  generated_by TEXT NOT NULL
);

-- Simple partition seeding for analytics tables
DO $$
DECLARE
  bases TEXT[] := ARRAY['usage_metrics','performance_metrics'];
  base TEXT; s1 DATE := date_trunc('month', now())::date; s2 DATE := (date_trunc('month', now()) + INTERVAL '1 month')::date;
  e1 DATE := (s1 + INTERVAL '1 month')::date; e2 DATE := (s2 + INTERVAL '1 month')::date;
  p1 TEXT; p2 TEXT;
BEGIN
  FOREACH base IN ARRAY bases LOOP
    p1 := format('%s_%s', base, to_char(s1,'YYYY_MM'));
    p2 := format('%s_%s', base, to_char(s2,'YYYY_MM'));
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L);', p1, base, s1, e1);
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L);', p2, base, s2, e2);
  END LOOP;
END $$;

-- ============ Indexes ============
-- Rulepacks
CREATE INDEX idx_rulepacks_active_tenant ON rulepacks(tenant_id, is_active) WHERE is_active = true;
CREATE INDEX idx_rulepacks_rules_gin ON rulepacks USING GIN (rules);
CREATE INDEX idx_rulepacks_metadata_gin ON rulepacks USING GIN (metadata);
CREATE INDEX idx_rulepacks_active_priority ON rulepacks(priority, tenant_id) WHERE is_active = true AND status='active';

-- Single active version per rulepack
CREATE UNIQUE INDEX idx_rulepack_versions_single_active ON rulepack_versions(rulepack_id) WHERE status='active';
CREATE INDEX idx_rulepack_versions_dsl_gin ON rulepack_versions USING GIN (dsl);

-- Assignments
CREATE INDEX idx_assignments_tenant_priority_enabled ON rulepack_assignments(tenant_id, priority DESC) WHERE enabled = true;
CREATE INDEX idx_assignments_tenant_scope ON rulepack_assignments(tenant_id, target_scope, enabled) WHERE enabled = true;
CREATE INDEX idx_assignments_trgm_scope ON rulepack_assignments USING GIN (target_scope gin_trgm_ops);

-- Ops tables
CREATE INDEX idx_violations_tenant_date_severity ON violations (tenant_id, created_at DESC, severity);
CREATE INDEX idx_scan_results_tenant_endpoint_date ON scan_results (tenant_id, endpoint, created_at DESC);
CREATE INDEX brin_violations_created_at ON violations USING BRIN (created_at);
CREATE INDEX brin_scan_results_created_at ON scan_results USING BRIN (created_at);
CREATE INDEX brin_audits_created_at ON audits USING BRIN (created_at);

-- Analytics
CREATE INDEX brin_usage_metrics_created_at ON usage_metrics USING BRIN (created_at);
CREATE INDEX brin_performance_metrics_created_at ON performance_metrics USING BRIN (created_at);

-- Tools
CREATE INDEX idx_tools_tenant_toolid ON tools(tenant_id, tool_id);
CREATE INDEX idx_tools_capability_tags_gin ON tools USING GIN (capability_tags);
CREATE INDEX idx_tools_data_domains_gin ON tools USING GIN (data_domains);

-- Tenant settings
CREATE INDEX idx_tenant_settings_tool_policies ON tenant_settings(tenant_id, key) WHERE key='tool_policies';

-- Billing indexes
CREATE INDEX idx_subscriptions_tenant_id ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_stripe_customer ON subscriptions(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;
CREATE INDEX idx_llm_usage_tenant_date ON llm_usage(tenant_id, usage_date);
CREATE INDEX idx_usage_billing_tenant_period ON usage_billing(tenant_id, billing_period_start, billing_period_end);
CREATE INDEX idx_usage_billing_status ON usage_billing(status);
CREATE INDEX idx_usage_billing_stripe_invoice ON usage_billing(stripe_invoice_id) WHERE stripe_invoice_id IS NOT NULL;

-- Invoice indexes
CREATE INDEX idx_invoices_tenant_id ON invoices(tenant_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_billing_period ON invoices(billing_period_start, billing_period_end);
CREATE INDEX idx_invoices_stripe_invoice ON invoices(stripe_invoice_id) WHERE stripe_invoice_id IS NOT NULL;
CREATE INDEX idx_invoice_line_items_invoice_id ON invoice_line_items(invoice_id);
CREATE INDEX idx_invoice_line_items_line_type ON invoice_line_items(line_type);

-- Experiment indexes
CREATE INDEX idx_experiments_status ON experiments(status);
CREATE INDEX idx_experiments_type ON experiments(type);
CREATE INDEX idx_experiments_dates ON experiments(start_date, end_date);
CREATE INDEX idx_experiment_variants_experiment_id ON experiment_variants(experiment_id);
CREATE INDEX idx_experiment_variants_type ON experiment_variants(type);
CREATE INDEX idx_experiment_assignments_user_tenant ON experiment_assignments(user_id, tenant_id);
CREATE INDEX idx_experiment_assignments_experiment ON experiment_assignments(experiment_id);
CREATE INDEX idx_experiment_assignments_variant ON experiment_assignments(variant_id);
CREATE INDEX idx_experiment_events_assignment ON experiment_events(assignment_id);
CREATE INDEX idx_experiment_events_name ON experiment_events(event_name);
CREATE INDEX idx_experiment_events_timestamp ON experiment_events(timestamp);

-- ============ RLS ============
ALTER TABLE rulepacks ENABLE ROW LEVEL SECURITY;
ALTER TABLE rulepack_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE rulepack_assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE tools ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE scan_results ENABLE ROW LEVEL SECURITY;
ALTER TABLE audits ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE performance_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_evidence ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE llm_usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_billing ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoice_line_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE experiments ENABLE ROW LEVEL SECURITY;
ALTER TABLE experiment_variants ENABLE ROW LEVEL SECURITY;
ALTER TABLE experiment_assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE experiment_events ENABLE ROW LEVEL SECURITY;

DO $$
DECLARE t TEXT;
BEGIN
  FOREACH t IN ARRAY ARRAY['rulepacks','rulepack_versions','rulepack_assignments','tools','tenant_settings','violations','scan_results','audits','usage_metrics','performance_metrics','compliance_evidence','subscriptions','llm_usage','usage_billing','invoices','invoice_line_items','experiments','experiment_variants','experiment_assignments','experiment_events'] LOOP
    EXECUTE format('DROP POLICY IF EXISTS tenant_isolation_policy ON %I;', t);
    EXECUTE format('CREATE POLICY tenant_isolation_policy ON %I FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());', t);
  END LOOP;
END $$;

-- ============ Billing Helper Functions ============
CREATE OR REPLACE FUNCTION get_tenant_subscription(tenant_uuid UUID)
RETURNS TABLE (
  subscription_id UUID,
  plan_name TEXT,
  plan_display_name TEXT,
  status TEXT,
  billing_cycle TEXT,
  current_period_end TIMESTAMPTZ,
  limits JSONB
) AS $$
BEGIN
  RETURN QUERY
  SELECT 
    s.id,
    sp.name,
    sp.display_name,
    s.status,
    s.billing_cycle,
    s.current_period_end,
    sp.limits
  FROM subscriptions s
  JOIN subscription_plans sp ON s.plan_id = sp.id
  WHERE s.tenant_id = tenant_uuid
    AND s.status IN ('active', 'trial');
END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION record_llm_usage(tenant_uuid UUID, usage_date DATE DEFAULT CURRENT_DATE)
RETURNS VOID AS $$
BEGIN
  INSERT INTO llm_usage (tenant_id, usage_date, llm_calls_count)
  VALUES (tenant_uuid, usage_date, 1)
  ON CONFLICT (tenant_id, usage_date)
  DO UPDATE SET 
    llm_calls_count = llm_usage.llm_calls_count + 1,
    updated_at = now();
END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_llm_usage_for_period(tenant_uuid UUID, start_date DATE, end_date DATE)
RETURNS INTEGER AS $$
DECLARE
  total_usage INTEGER;
BEGIN
  SELECT COALESCE(SUM(llm_calls_count), 0)
  INTO total_usage
  FROM llm_usage
  WHERE tenant_id = tenant_uuid
    AND usage_date >= start_date
    AND usage_date <= end_date;
  
  RETURN total_usage;
END; $$ LANGUAGE plpgsql;

-- ============ Retention & Partitions (Optional Scheduling) ============
CREATE OR REPLACE FUNCTION manage_data_retention()
RETURNS VOID AS $$
BEGIN
  -- Example: keep violations 30 days hot
  DELETE FROM ONLY violations WHERE created_at < now() - INTERVAL '30 days';
  -- Keep scan_results 7 days hot
  DELETE FROM ONLY scan_results WHERE created_at < now() - INTERVAL '7 days';
END; $$ LANGUAGE plpgsql;

-- If pg_cron is available, schedule jobs (best effort)
DO $$ BEGIN
  PERFORM cron.schedule('data-retention-daily','0 2 * * *','SELECT manage_data_retention();');
EXCEPTION WHEN undefined_table THEN NULL; WHEN undefined_function THEN NULL; END $$;

