#!/bin/bash
set -e

echo "🚀 PromptShield Supabase Setup Script"
echo "======================================"

# Check if supabase CLI is installed
if ! command -v supabase &> /dev/null; then
    echo "Installing Supabase CLI..."
    npm install -g supabase
fi

echo ""
echo "📋 Setup Instructions:"
echo "1. Go to https://supabase.com and create a free account"
echo "2. Create a new project (choose region closest to your users)"
echo "3. Copy your connection details from Project Settings > Database"
echo "4. Run this script with your connection string"
echo ""

# Check if connection string provided
if [ -z "$1" ]; then
    echo "Usage: $0 'postgres://postgres:[password]@[host]:5432/postgres'"
    echo ""
    echo "Example:"
    echo "  $0 'postgres://postgres:yourpassword@db.yourproject.supabase.co:5432/postgres'"
    exit 1
fi

SUPABASE_URL="$1"
echo "🔌 Testing connection to: $SUPABASE_URL"

# Test connection
if psql "$SUPABASE_URL" -c "SELECT version();" > /dev/null 2>&1; then
    echo "✅ Database connection successful!"
else
    echo "❌ Database connection failed. Please check your connection string."
    exit 1
fi

echo ""
echo "🗄️  Running migrations..."

# Run our migration
psql "$SUPABASE_URL" -f migrations/0001_init.sql

echo ""
echo "👤 Creating sample tenant..."

# Create a sample tenant
psql "$SUPABASE_URL" -c "
INSERT INTO tenants (id, name) 
VALUES (gen_random_uuid(), 'default-tenant') 
ON CONFLICT (name) DO NOTHING;
"

TENANT_ID=$(psql "$SUPABASE_URL" -t -c "SELECT id FROM tenants WHERE name = 'default-tenant';")
TENANT_ID=$(echo $TENANT_ID | xargs) # trim whitespace

echo "✅ Created tenant: $TENANT_ID"

echo ""
echo "📝 Environment Configuration:"
echo "Add these to your .env file:"
echo ""
echo "export PS_PG_DSN=\"$SUPABASE_URL\""
echo "export PS_TENANT_ID=\"$TENANT_ID\""
echo "export PS_CONTROL_PLANE_ADDR=\":8085\""
echo ""

echo "🎉 Setup complete! You can now start the control plane:"
echo "  export PS_PG_DSN=\"$SUPABASE_URL\""
echo "  ./bin/ps-controlplane"