param(
  [string]$Profile = $env:AWS_PROFILE,
  [string]$Region = $(if ($env:AURORA_REGION) { $env:AURORA_REGION } else { 'us-east-1' }),
  [string]$SecretId = $env:AURORA_SECRET_ID,
  [string]$Writer   = $env:AURORA_WRITER,
  [string]$DbName   = $(if ($env:AURORA_DB_NAME) { $env:AURORA_DB_NAME } else { 'promptshield' })
)

$ErrorActionPreference = 'Stop'

if (-not $Profile) { $Profile = 'default' }
$env:AWS_PROFILE = $Profile
$env:AWS_REGION  = $Region
$env:AWS_SDK_LOAD_CONFIG = '1'

if (-not $SecretId) { throw "SecretId is required (set -SecretId or AURORA_SECRET_ID)." }
if (-not $Writer)   { throw "Writer endpoint is required (set -Writer or AURORA_WRITER)." }

Write-Host "Using profile=$Profile region=$Region writer=$Writer db=$DbName"

# Fetch DB credentials from Secrets Manager without printing them
$secJson = aws secretsmanager get-secret-value --region $Region --secret-id $SecretId --query SecretString --output text
if (-not $secJson) { throw "Failed to retrieve secret: $SecretId" }
$creds = $secJson | ConvertFrom-Json

$env:PGUSER = $creds.username
$env:PGPASSWORD = $creds.password

# SQL to verify schema, RLS flags, and policies; also do a simple RLS behavior check
$sql = @"
SELECT 'db:'||current_database();
SELECT 'server_version:'||version();

SELECT 'tables:';
SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name;

SELECT E'\nrls:';
SELECT relname||':'||relrowsecurity FROM pg_class WHERE relnamespace='public'::regnamespace AND relkind='r' ORDER BY relname;

SELECT E'\npolicies:';
SELECT polname||':'||polrelid::regclass FROM pg_policy ORDER BY polrelid::regclass::text, polname;

-- RLS context test: show visibility with and without tenant context
SELECT set_config('app.current_tenant_id', NULL, false);
SELECT 'tenant_ctx:'||COALESCE(current_setting('app.current_tenant_id', true),'NULL');
SELECT 'rulepacks_visible_no_ctx:'||(SELECT count(*) FROM rulepacks);

-- Set any tenant context (first tenant) and re-check visibility
SELECT set_config('app.current_tenant_id', (SELECT id::text FROM tenants ORDER BY created_at LIMIT 1), false);
SELECT 'tenant_ctx_after_set:'||current_setting('app.current_tenant_id', true);
SELECT 'rulepacks_visible_with_ctx:'||(SELECT count(*) FROM rulepacks);
"@

$sql | psql "host=$Writer dbname=$DbName sslmode=require" -At -v ON_ERROR_STOP=1 -P pager=off

Write-Host "Verification complete."
