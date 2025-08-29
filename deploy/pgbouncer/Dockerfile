FROM pgbouncer/pgbouncer:1.21.0

# Install required packages
USER root
RUN apk add --no-cache \
    ca-certificates \
    curl \
    postgresql-client

# Copy configuration files
COPY pgbouncer.ini /etc/pgbouncer/pgbouncer.ini.template
COPY docker-entrypoint.sh /docker-entrypoint.sh

# Create necessary directories
RUN mkdir -p /var/log/pgbouncer /var/run/pgbouncer && \
    chown -R pgbouncer:pgbouncer /var/log/pgbouncer /var/run/pgbouncer && \
    chmod +x /docker-entrypoint.sh

# Health check script
COPY healthcheck.sh /usr/local/bin/healthcheck.sh
RUN chmod +x /usr/local/bin/healthcheck.sh

USER pgbouncer

EXPOSE 6432

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
    CMD /usr/local/bin/healthcheck.sh || exit 1

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["pgbouncer", "/etc/pgbouncer/pgbouncer.ini"]