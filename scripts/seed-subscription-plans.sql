-- Seed subscription plans for PromptShield
-- This script creates the default subscription plans based on our premium pricing strategy

-- Professional Plan: $299/month
INSERT INTO subscription_plans (
    id, name, display_name, description, price_monthly, price_yearly,
    features, limits, is_active, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'professional',
    'Professional',
    'Perfect for mid-market companies (50-500 employees) with comprehensive AI security needs',
    29900, -- $299.00 in cents
    299000, -- $2,990.00 yearly (2 months free)
    '{
        "unlimited_rulepacks": true,
        "compliance_reporting": ["SOC2", "GDPR"],
        "support_level": "email",
        "sla": "99.5%",
        "custom_integrations": false,
        "advanced_analytics": true,
        "air_gapped_deployment": false,
        "priority_processing": false,
        "white_labeling": false,
        "sso": true,
        "audit_logs": true,
        "custom_retention": false
    }'::jsonb,
    '{
        "api_calls_monthly": 1000000,
        "llm_calls_monthly": 10000,
        "rulepacks": -1,
        "users": 100,
        "data_retention_days": 90
    }'::jsonb,
    true,
    now(),
    now()
);

-- Enterprise Plan: $1,999/month
INSERT INTO subscription_plans (
    id, name, display_name, description, price_monthly, price_yearly,
    features, limits, is_active, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'enterprise',
    'Enterprise',
    'For large enterprises (500+ employees) with advanced compliance and security requirements',
    199900, -- $1,999.00 in cents
    1999000, -- $19,990.00 yearly (2 months free)
    '{
        "unlimited_rulepacks": true,
        "compliance_reporting": ["SOC2", "GDPR", "HIPAA", "NIST"],
        "support_level": "24/7",
        "sla": "99.9%",
        "custom_integrations": true,
        "advanced_analytics": true,
        "air_gapped_deployment": true,
        "priority_processing": true,
        "white_labeling": true,
        "sso": true,
        "audit_logs": true,
        "custom_retention": true
    }'::jsonb,
    '{
        "api_calls_monthly": 10000000,
        "llm_calls_monthly": 100000,
        "rulepacks": -1,
        "users": 1000,
        "data_retention_days": 365
    }'::jsonb,
    true,
    now(),
    now()
);

-- Enterprise Plus Plan: Custom pricing
INSERT INTO subscription_plans (
    id, name, display_name, description, price_monthly, price_yearly,
    features, limits, is_active, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'enterprise_plus',
    'Enterprise Plus',
    'Custom enterprise solution for organizations with unique requirements and high-volume usage',
    0, -- Custom pricing
    0, -- Custom pricing
    '{
        "unlimited_rulepacks": true,
        "compliance_reporting": ["SOC2", "GDPR", "HIPAA", "NIST", "PCI-DSS", "ISO27001"],
        "support_level": "24/7",
        "sla": "99.99%",
        "custom_integrations": true,
        "advanced_analytics": true,
        "air_gapped_deployment": true,
        "priority_processing": true,
        "white_labeling": true,
        "sso": true,
        "audit_logs": true,
        "custom_retention": true
    }'::jsonb,
    '{
        "api_calls_monthly": -1,
        "llm_calls_monthly": -1,
        "rulepacks": -1,
        "users": -1,
        "data_retention_days": -1
    }'::jsonb,
    true,
    now(),
    now()
);

-- Verify the plans were created
SELECT 
    name,
    display_name,
    price_monthly,
    features->>'support_level' as support_level,
    features->>'sla' as sla,
    limits->>'llm_calls_monthly' as llm_calls_included
FROM subscription_plans 
WHERE is_active = true 
ORDER BY price_monthly ASC;
