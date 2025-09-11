-- Tenant services and time-series metrics (partitioned)

CREATE TABLE IF NOT EXISTS tenant_services (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  service_name TEXT NOT NULL,
  service_type TEXT NOT NULL DEFAULT 'enforcer' CHECK (service_type IN ('enforcer','scanner','analyzer','proxy')),
  status TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('stopped','starting','running','stopping','error','partial')),
  enabled BOOLEAN DEFAULT true,
  config JSONB NOT NULL DEFAULT '{}'::jsonb,
  environment JSONB NOT NULL DEFAULT '{}'::jsonb,
  resources JSONB NOT NULL DEFAULT '{"cpu":"500m","memory":"512Mi","replicas":1}'::jsonb,
  deployment_id TEXT,
  namespace TEXT DEFAULT 'promptshield',
  version TEXT DEFAULT 'latest',
  last_started TIMESTAMPTZ,
  last_stopped TIMESTAMPTZ,
  last_health_check TIMESTAMPTZ,
  health_status TEXT DEFAULT 'unknown' CHECK (health_status IN ('healthy','unhealthy','unknown')),
  error_message TEXT,
  uptime_seconds INTEGER DEFAULT 0,
  request_count BIGINT DEFAULT 0,
  error_count BIGINT DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  UNIQUE(tenant_id, service_name)
);

CREATE TABLE IF NOT EXISTS service_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  service_id UUID REFERENCES tenant_services(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL CHECK (event_type IN (
    'service.created','service.started','service.stopped',
    'service.restarted','service.scaled','service.updated',
    'service.error','service.health.changed','service.deleted','service.status.changed'
  )),
  severity TEXT DEFAULT 'info' CHECK (severity IN ('debug','info','warning','error','critical')),
  message TEXT,
  details JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Partitioned service metrics (by month)
CREATE TABLE IF NOT EXISTS service_metrics (
  id UUID NOT NULL DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  service_id UUID NOT NULL,
  metric_name TEXT NOT NULL,
  metric_value DECIMAL(15,4) NOT NULL,
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (recorded_at);

-- Create current and next month partitions dynamically
DO $$
DECLARE
  start_date date := date_trunc('month', now())::date;
  next_date  date := (date_trunc('month', now()) + interval '1 month')::date;
  part_name  text;
BEGIN
  part_name := 'service_metrics_' || to_char(start_date, 'YYYY_MM');
  EXECUTE format(
    'CREATE TABLE IF NOT EXISTS %I PARTITION OF service_metrics FOR VALUES FROM (%L) TO (%L)',
    part_name, start_date, (start_date + interval '1 month')::date
  );

  part_name := 'service_metrics_' || to_char(next_date, 'YYYY_MM');
  EXECUTE format(
    'CREATE TABLE IF NOT EXISTS %I PARTITION OF service_metrics FOR VALUES FROM (%L) TO (%L)',
    part_name, next_date, (next_date + interval '1 month')::date
  );
END $$;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_tenant_services_tenant_id ON tenant_services(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_services_status ON tenant_services(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_tenant_services_type ON tenant_services(service_type, status);
CREATE INDEX IF NOT EXISTS idx_tenant_services_health ON tenant_services(health_status, last_health_check);
CREATE INDEX IF NOT EXISTS idx_tenant_services_active ON tenant_services(tenant_id, enabled) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_service_events_tenant_id ON service_events(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_events_service_id ON service_events(service_id, created_at DESC);

-- Partitioned indexes
CREATE INDEX IF NOT EXISTS idx_service_metrics_tenant_time ON service_metrics(tenant_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_metrics_service_time ON service_metrics(service_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_metrics_name_time ON service_metrics(metric_name, recorded_at DESC);

