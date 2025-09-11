#!/bin/sh
set -e

# Generate pgbouncer.ini from template with environment variables
envsubst < /etc/pgbouncer/pgbouncer.ini.template > /etc/pgbouncer/pgbouncer.ini

# Generate userlist.txt for authentication
# Format: "username" "password_hash"
cat > /etc/pgbouncer/userlist.txt << EOF
"${PS_DB_USER:-postgres}" "md5$(echo -n "${PS_DB_PASSWORD}${PS_DB_USER:-postgres}" | md5sum | cut -d' ' -f1)"
"pgbouncer" "md5$(echo -n "${PGBOUNCER_ADMIN_PASSWORD:-admin}pgbouncer" | md5sum | cut -d' ' -f1)"
"monitor" "md5$(echo -n "${PGBOUNCER_MONITOR_PASSWORD:-monitor}monitor" | md5sum | cut -d' ' -f1)"
EOF

# Create pgbouncer database for admin connections
cat > /etc/pgbouncer/pgbouncer_db.ini << EOF
[databases]
pgbouncer = host=127.0.0.1 port=6432 dbname=pgbouncer auth_user=pgbouncer
EOF

# Log startup
echo "Starting PgBouncer with configuration:"
echo "  Supabase Host: ${SUPABASE_HOST}"
echo "  Pool Mode: transaction"
echo "  Max Client Connections: 1000"
echo "  Default Pool Size: 25"
echo "  Max DB Connections: 100"

# Start PgBouncer
exec "$@"