#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SSL_DIR="./deploy/ssl"
DOMAINS=("api.promptshield.com" "proxy.promptshield.com")
CERT_VALIDITY=365

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Create SSL directory structure
create_ssl_dirs() {
    log_info "Creating SSL directory structure..."
    for domain in "${DOMAINS[@]}"; do
        mkdir -p "${SSL_DIR}/${domain}"
    done
}

# Generate self-signed certificates for development
generate_dev_certs() {
    log_warn "Generating self-signed certificates for DEVELOPMENT use only"
    
    for domain in "${DOMAINS[@]}"; do
        log_info "Generating certificate for ${domain}..."
        
        # Generate private key
        openssl genrsa -out "${SSL_DIR}/${domain}/privkey.pem" 2048
        
        # Generate certificate signing request
        openssl req -new \
            -key "${SSL_DIR}/${domain}/privkey.pem" \
            -out "${SSL_DIR}/${domain}/csr.pem" \
            -subj "/C=US/ST=State/L=City/O=PromptShield/CN=${domain}"
        
        # Generate self-signed certificate
        openssl x509 -req \
            -days ${CERT_VALIDITY} \
            -in "${SSL_DIR}/${domain}/csr.pem" \
            -signkey "${SSL_DIR}/${domain}/privkey.pem" \
            -out "${SSL_DIR}/${domain}/cert.pem"
        
        # Create fullchain (for self-signed, it's just the cert)
        cp "${SSL_DIR}/${domain}/cert.pem" "${SSL_DIR}/${domain}/fullchain.pem"
        
        # Create chain (empty for self-signed)
        touch "${SSL_DIR}/${domain}/chain.pem"
        
        # Set proper permissions
        chmod 600 "${SSL_DIR}/${domain}/privkey.pem"
        chmod 644 "${SSL_DIR}/${domain}/fullchain.pem"
        chmod 644 "${SSL_DIR}/${domain}/cert.pem"
        
        log_info "Certificate generated for ${domain}"
    done
}

# Generate Let's Encrypt certificates for production
generate_prod_certs() {
    log_info "Setting up Let's Encrypt certificates for PRODUCTION"
    
    # Check if certbot is installed
    if ! command -v certbot &> /dev/null; then
        log_warn "Certbot is not installed. Installing..."
        if [ -f /etc/debian_version ]; then
            sudo apt-get update
            sudo apt-get install -y certbot
        elif [ -f /etc/redhat-release ]; then
            sudo yum install -y certbot
        else
            log_warn "Please install certbot manually"
            exit 1
        fi
    fi
    
    # Create webroot for ACME challenges
    mkdir -p /var/www/certbot
    
    for domain in "${DOMAINS[@]}"; do
        log_info "Requesting certificate for ${domain}..."
        
        certbot certonly \
            --webroot \
            --webroot-path=/var/www/certbot \
            --email admin@promptshield.com \
            --agree-tos \
            --no-eff-email \
            --force-renewal \
            -d ${domain}
        
        # Symlink to our SSL directory
        ln -sf "/etc/letsencrypt/live/${domain}/privkey.pem" "${SSL_DIR}/${domain}/privkey.pem"
        ln -sf "/etc/letsencrypt/live/${domain}/fullchain.pem" "${SSL_DIR}/${domain}/fullchain.pem"
        ln -sf "/etc/letsencrypt/live/${domain}/chain.pem" "${SSL_DIR}/${domain}/chain.pem"
    done
}

# Setup certificate renewal
setup_renewal() {
    log_info "Setting up automatic certificate renewal..."
    
    # Create renewal script
    cat > /etc/cron.daily/certbot-renew << 'EOF'
#!/bin/bash
certbot renew --quiet --post-hook "docker exec nginx nginx -s reload"
EOF
    
    chmod +x /etc/cron.daily/certbot-renew
    log_info "Automatic renewal configured"
}

# Main script
main() {
    ENVIRONMENT=${1:-development}
    
    create_ssl_dirs
    
    if [ "$ENVIRONMENT" = "production" ]; then
        generate_prod_certs
        setup_renewal
    else
        generate_dev_certs
    fi
    
    log_info "SSL certificates setup complete!"
    log_info "Certificate locations:"
    for domain in "${DOMAINS[@]}"; do
        echo "  ${domain}:"
        echo "    Private Key: ${SSL_DIR}/${domain}/privkey.pem"
        echo "    Certificate: ${SSL_DIR}/${domain}/fullchain.pem"
    done
    
    if [ "$ENVIRONMENT" = "development" ]; then
        log_warn "⚠️  Using self-signed certificates - browsers will show security warnings"
        log_warn "Add certificates to trusted store or use --insecure flag with curl"
    fi
}

# Check for required tools
if ! command -v openssl &> /dev/null; then
    echo "Error: openssl is required but not installed"
    exit 1
fi

main "$@"