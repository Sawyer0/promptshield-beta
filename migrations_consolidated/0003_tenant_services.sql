-- ================================================================
-- TENANT SERVICES MANAGEMENT - FROM 0008
-- ================================================================
-- Creates: Service lifecycle management tables
-- Purpose: Track and manage tenant-specific service deployments
-- Date: Consolidated 2025-08-27

-- ================================================================
-- SERVICE MANAGEMENT TABLES
-- ================================================================

-- 1. TENANT SERVICES (Service instances per tenant)
CREATE TABLE tenant_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_name TEXT NOT NULL,
    service_type TEXT NOT NULL DEFAULT 'enforcer' CHECK (service_type IN ('enforcer', 'scanner', 'analyzer', 'proxy')),
    status TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('stopped', 'starting', 'running', 'stopping', 'error', 'partial')),
    enabled BOOLEAN DEFAULT true,
    
    -- Configuration
    config JSONB DEFAULT '{}',
    environment JSONB DEFAULT '{}',
    resources JSONB DEFAULT '{"cpu": "500m", "memory": "512Mi", "replicas": 1}',
    
    -- Deployment info
    deployment_id TEXT,
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

-- 2. SERVICE EVENTS (Event log for service lifecycle)
CREATE TABLE service_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_id UUID REFERENCES tenant_services(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN (
        'service.created', 'service.started', 'service.stopped', 
        'service.restarted', 'service.scaled', 'service.updated',
        'service.error', 'service.health.changed', 'service.deleted'
    )),
    severity TEXT DEFAULT 'info' CHECK (severity IN ('debug', 'info', 'warning', 'error', 'critical')),
    message TEXT,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. SERVICE METRICS (Time-series metrics for services)
CREATE TABLE service_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES tenant_services(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    metric_value DECIMAL(15,4) NOT NULL,
    labels JSONB DEFAULT '{}',
    recorded_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Partition by time for efficient queries
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ================================================================
-- INDEXES FOR SERVICE TABLES
-- ================================================================

-- Tenant Services indexes
CREATE INDEX idx_tenant_services_tenant_id ON tenant_services(tenant_id);
CREATE INDEX idx_tenant_services_status ON tenant_services(tenant_id, status);
CREATE INDEX idx_tenant_services_type ON tenant_services(service_type, status);
CREATE INDEX idx_tenant_services_health ON tenant_services(health_status, last_health_check);
CREATE INDEX idx_tenant_services_active ON tenant_services(tenant_id, enabled) WHERE deleted_at IS NULL;

-- Service Events indexes
CREATE INDEX idx_service_events_tenant_id ON service_events(tenant_id, created_at DESC);
CREATE INDEX idx_service_events_service_id ON service_events(service_id, created_at DESC);
CREATE INDEX idx_service_events_type ON service_events(event_type, created_at DESC);
CREATE INDEX idx_service_events_severity ON service_events(severity, created_at DESC);

-- Service Metrics indexes (optimized for time-series queries)
CREATE INDEX idx_service_metrics_tenant_time ON service_metrics(tenant_id, recorded_at DESC);
CREATE INDEX idx_service_metrics_service_time ON service_metrics(service_id, recorded_at DESC);
CREATE INDEX idx_service_metrics_name_time ON service_metrics(metric_name, recorded_at DESC);
CREATE INDEX idx_service_metrics_composite ON service_metrics(service_id, metric_name, recorded_at DESC);

-- ================================================================
-- FUNCTIONS FOR SERVICE MANAGEMENT
-- ================================================================

-- Function to update service status and log event
CREATE OR REPLACE FUNCTION update_service_status(
    p_service_id UUID,
    p_status TEXT,
    p_message TEXT DEFAULT NULL
) RETURNS VOID AS $$
DECLARE
    v_tenant_id UUID;
    v_old_status TEXT;
BEGIN
    -- Get tenant_id and current status
    SELECT tenant_id, status INTO v_tenant_id, v_old_status 
    FROM tenant_services 
    WHERE id = p_service_id;
    
    -- Update service status
    UPDATE tenant_services 
    SET 
        status = p_status,
        updated_at = NOW(),
        last_started = CASE WHEN p_status = 'running' THEN NOW() ELSE last_started END,
        last_stopped = CASE WHEN p_status = 'stopped' THEN NOW() ELSE last_stopped END,
        error_message = CASE WHEN p_status = 'error' THEN p_message ELSE NULL END
    WHERE id = p_service_id;
    
    -- Log the event if status changed
    IF v_old_status IS DISTINCT FROM p_status THEN
        INSERT INTO service_events (tenant_id, service_id, event_type, message)
        VALUES (
            v_tenant_id, 
            p_service_id, 
            'service.status.changed',
            COALESCE(p_message, format('Service status changed from %s to %s', v_old_status, p_status))
        );
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to record service metrics
CREATE OR REPLACE FUNCTION record_service_metric(
    p_service_id UUID,
    p_metric_name TEXT,
    p_metric_value DECIMAL,
    p_labels JSONB DEFAULT '{}'
) RETURNS VOID AS $$
DECLARE
    v_tenant_id UUID;
BEGIN
    -- Get tenant_id
    SELECT tenant_id INTO v_tenant_id 
    FROM tenant_services 
    WHERE id = p_service_id;
    
    -- Insert metric
    INSERT INTO service_metrics (tenant_id, service_id, metric_name, metric_value, labels)
    VALUES (v_tenant_id, p_service_id, p_metric_name, p_metric_value, p_labels);
END;
$$ LANGUAGE plpgsql;

-- ================================================================
-- COMMENTS FOR DOCUMENTATION
-- ================================================================

COMMENT ON TABLE tenant_services IS 'Service instances deployed for each tenant';
COMMENT ON TABLE service_events IS 'Event log for service lifecycle operations';  
COMMENT ON TABLE service_metrics IS 'Time-series metrics for service monitoring';
COMMENT ON FUNCTION update_service_status(UUID, TEXT, TEXT) IS 'Updates service status and logs event';
COMMENT ON FUNCTION record_service_metric(UUID, TEXT, DECIMAL, JSONB) IS 'Records service metrics for monitoring';