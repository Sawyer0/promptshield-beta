-- Rulepack names are unique per tenant. The service checks before inserting;
-- this index closes the race between concurrent writers in different processes.
-- Rename or delete existing duplicates before applying.
CREATE UNIQUE INDEX IF NOT EXISTS rulepacks_tenant_name_key ON rulepacks (tenant_id, name);
