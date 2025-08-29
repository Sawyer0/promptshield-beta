-- Tenant Services Management
-- This table tracks services deployed for each tenant

CREATE TABLE IF NOT EXISTS tenant_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_name TEXT NOT NULL,
    service_type TEXT NOT NULL DEFAULT 'enforcer' CHECK (service_type IN ('enforcer', 'scanner', 'analyzer', 'proxy')),
    status TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('stopped', 'starting', 'running', 'stopping', 'error', 'partial')),
    enabled BOOLEAN DEFAULT true,
    
    -- Configuration
    config JSONB DEFAULT '{}',
    environment JSONB DEFAULT '{}', -- Environment variables
    resources JSONB DEFAULT '{"cpu": "500m", "memory": "512Mi", "replicas": 1}',
    
    -- Deployment info
    deployment_id TEXT, -- K8s deployment name or Docker container ID
    namespace TEXT DEFAULT 'promptshield',
    version TEXT DEFAULT 'latest',
    
    -- Status tracking
    last_started TIMESTAMPTZ,
    last_stopped TIMESTAMPTZ,
    last_health_check TIMESTAMPTZ,
    health_status TEXT DEFAULT 'unknown' CHECK (health_status IN ('healthy', 'unhealthy', 'unknown')),
    error_message TEXT,
    
    -- Metrics
    uptime_seconds INTEGER DEFAULT 0,
    request_count BIGINT DEFAULT 0,
    error_count BIGINT DEFAULT 0,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    UNIQUE(tenant_id, service_name)
);

-- Service Events Log
CREATE TABLE IF NOT EXISTS service_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_id UUID REFERENCES tenant_services(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN (
        'service.created', 'service.started', 'service.stopped', 
        'service.restarted', 'service.scaled', 'service.updated',
        'service.error', 'service.health.changed'
    )),
    severity TEXT NOT NULL DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    message TEXT,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Service Metrics (time-series data)
CREATE TABLE IF NOT EXISTS service_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES tenant_services(id) ON DELETE CASCADE,
    metric_type TEXT NOT NULL CHECK (metric_type IN ('cpu', 'memory', 'requests', 'errors', 'latency')),
    value DECIMAL NOT NULL,
    unit TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB
);

-- Indexes for performance
CREATE INDEX idx_tenant_services_tenant ON tenant_services(tenant_id);
CREATE INDEX idx_tenant_services_status ON tenant_services(status);
CREATE INDEX idx_tenant_services_enabled ON tenant_services(enabled) WHERE enabled = true;
CREATE INDEX idx_service_events_tenant ON service_events(tenant_id);
CREATE INDEX idx_service_events_service ON service_events(service_id);
CREATE INDEX idx_service_events_created ON service_events(created_at DESC);
CREATE INDEX idx_service_metrics_service_time ON service_metrics(service_id, timestamp DESC);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_tenant_services_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tenant_services_updated_at_trigger
    BEFORE UPDATE ON tenant_services
    FOR EACH ROW
    EXECUTE FUNCTION update_tenant_services_updated_at();

-- Insert default service configurations for existing tenants
INSERT INTO tenant_services (tenant_id, service_name, service_type, status, config)
SELECT 
    id as tenant_id,
    'default-enforcer' as service_name,
    'enforcer' as service_type,
    'stopped' as status,
    jsonb_build_object(
        'mode', 'enforce',
        'timeout', '300ms',
        'max_body_bytes', 10485760,
        'fail_on', 'HIGH',
        'rulepack', 'default'
    ) as config
FROM tenants
WHERE NOT EXISTS (
    SELECT 1 FROM tenant_services ts 
    WHERE ts.tenant_id = tenants.id 
    AND ts.service_name = 'default-enforcer'
);

-- Row Level Security
ALTER TABLE tenant_services ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_metrics ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY "Tenants can view own services" ON tenant_services
    FOR SELECT USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE POLICY "Tenants can manage own services" ON tenant_services
    FOR ALL USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE POLICY "Tenants can view own service events" ON service_events
    FOR SELECT USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE POLICY "Tenants can view own service metrics" ON service_metrics
    FOR SELECT USING (
        service_id IN (
            SELECT id FROM tenant_services 
            WHERE tenant_id = current_setting('app.current_tenant')::uuid
        )
    );

-- Comments
COMMENT ON TABLE tenant_services IS 'Manages deployed services per tenant with configuration and status tracking';
COMMENT ON TABLE service_events IS 'Audit log of service lifecycle events';
COMMENT ON TABLE service_metrics IS 'Time-series metrics data for service monitoring';
COMMENT ON COLUMN tenant_services.status IS 'Current service status: stopped, starting, running, stopping, error, partial';
COMMENT ON COLUMN tenant_services.config IS 'Service-specific configuration as JSON';
COMMENT ON COLUMN tenant_services.resources IS 'Resource limits: cpu, memory, replicas';