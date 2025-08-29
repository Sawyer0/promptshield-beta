#!/bin/sh

# Check if PgBouncer is responding
PGPASSWORD="${PGBOUNCER_MONITOR_PASSWORD:-monitor}" psql -h 127.0.0.1 -p 6432 -U monitor -d pgbouncer -c "SHOW POOLS;" > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo "PgBouncer is healthy"
    exit 0
else
    echo "PgBouncer health check failed"
    exit 1
fi