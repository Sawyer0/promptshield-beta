-- 0007_snapshot_refresh_triggers.sql
-- DB trigger to automatically refresh endpoint snapshots on assignment changes

CREATE OR REPLACE FUNCTION on_assignment_change_refresh()
RETURNS trigger AS $$
BEGIN
  IF (TG_OP = 'INSERT' OR TG_OP = 'UPDATE') THEN
    PERFORM refresh_endpoint_snapshots(NEW.tenant_id);
  ELSIF (TG_OP = 'DELETE') THEN
    PERFORM refresh_endpoint_snapshots(OLD.tenant_id);
  END IF;
  RETURN NULL;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_assignments_refresh ON rulepack_assignments;
CREATE TRIGGER trg_assignments_refresh
AFTER INSERT OR UPDATE OR DELETE ON rulepack_assignments
FOR EACH ROW EXECUTE PROCEDURE on_assignment_change_refresh();

