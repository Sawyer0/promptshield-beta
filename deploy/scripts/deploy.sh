#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DEPLOYMENT_TYPE=${1:-docker}
ENVIRONMENT=${2:-production}
NAMESPACE=promptshield

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        if ! command -v docker &> /dev/null; then
            log_error "Docker is not installed"
            exit 1
        fi
        if ! command -v docker compose &> /dev/null; then
            log_error "Docker Compose is not installed"
            exit 1
        fi
    elif [ "$DEPLOYMENT_TYPE" = "kubernetes" ]; then
        if ! command -v kubectl &> /dev/null; then
            log_error "kubectl is not installed"
            exit 1
        fi
        if ! kubectl cluster-info &> /dev/null; then
            log_error "No Kubernetes cluster connected"
            exit 1
        fi
    fi
    
    if [ ! -f ".env.${ENVIRONMENT}" ]; then
        log_error ".env.${ENVIRONMENT} file not found. Copy .env.production.example and configure it."
        exit 1
    fi
    
    log_info "Prerequisites check passed ✓"
}

deploy_docker() {
    log_info "🚀 Deploying to Docker Compose (${ENVIRONMENT})..."
    
    # Load environment
    export $(cat .env.${ENVIRONMENT} | grep -v '^#' | xargs)
    
    # Build images
    log_info "Building Docker images..."
    docker compose -f docker-compose.production.yaml build
    
    # Stop existing services
    log_info "Stopping existing services..."
    docker compose -f docker-compose.production.yaml down
    
    # Start services
    log_info "Starting services..."
    docker compose -f docker-compose.production.yaml up -d
    
    # Wait for services to be healthy
    log_info "Waiting for services to be healthy..."
    sleep 10
    
    # Health checks
    log_info "Running health checks..."
    
    if curl -sf http://localhost:9090/healthz > /dev/null; then
        log_info "✅ API is healthy"
    else
        log_error "❌ API health check failed"
        docker compose -f docker-compose.production.yaml logs promptshield-1
        exit 1
    fi
    
    if curl -sf http://localhost:8080 > /dev/null; then
        log_info "✅ Proxy is healthy"
    else
        log_error "❌ Proxy health check failed"
        docker compose -f docker-compose.production.yaml logs envoy-1
        exit 1
    fi
    
    log_info "🎉 Deployment successful!"
    log_info "API endpoint: http://localhost:9090"
    log_info "Proxy endpoint: http://localhost:8080"
    log_info "Prometheus: http://localhost:9092"
    log_info "Grafana: http://localhost:3000 (admin/admin)"
}

deploy_kubernetes() {
    log_info "🚀 Deploying to Kubernetes (${ENVIRONMENT})..."
    
    # Create namespace
    log_info "Creating namespace..."
    kubectl apply -f deploy/k8s/namespace.yaml
    
    # Create secrets from env file
    log_info "Creating secrets..."
    kubectl create secret generic promptshield-secrets \
        --from-env-file=.env.${ENVIRONMENT} \
        --namespace=${NAMESPACE} \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply configurations
    log_info "Applying configurations..."
    kubectl apply -f deploy/k8s/configmap.yaml
    kubectl apply -f deploy/k8s/rbac.yaml
    
    # Deploy applications
    log_info "Deploying applications..."
    kubectl apply -f deploy/k8s/deployment.yaml
    kubectl apply -f deploy/k8s/service.yaml
    kubectl apply -f deploy/k8s/hpa.yaml
    kubectl apply -f deploy/k8s/ingress.yaml
    
    # Wait for rollout
    log_info "Waiting for rollout to complete..."
    kubectl rollout status deployment/promptshield -n ${NAMESPACE} --timeout=5m
    kubectl rollout status deployment/envoy-proxy -n ${NAMESPACE} --timeout=5m
    
    # Get service endpoints
    log_info "Getting service endpoints..."
    kubectl get ingress -n ${NAMESPACE}
    kubectl get svc -n ${NAMESPACE}
    
    # Run health checks
    log_info "Running health checks..."
    POD=$(kubectl get pod -n ${NAMESPACE} -l app=promptshield -o jsonpath="{.items[0].metadata.name}")
    
    if kubectl exec -n ${NAMESPACE} ${POD} -- wget -qO- http://localhost:9090/healthz > /dev/null; then
        log_info "✅ PromptShield is healthy"
    else
        log_error "❌ PromptShield health check failed"
        kubectl logs -n ${NAMESPACE} ${POD}
        exit 1
    fi
    
    log_info "🎉 Deployment successful!"
}

rollback_docker() {
    log_info "Rolling back Docker deployment..."
    docker compose -f docker-compose.production.yaml down
    # Restore previous version if tagged
    if [ -n "$PREVIOUS_VERSION" ]; then
        docker compose -f docker-compose.production.yaml up -d
    fi
}

rollback_kubernetes() {
    log_info "Rolling back Kubernetes deployment..."
    kubectl rollout undo deployment/promptshield -n ${NAMESPACE}
    kubectl rollout undo deployment/envoy-proxy -n ${NAMESPACE}
    kubectl rollout status deployment/promptshield -n ${NAMESPACE}
    kubectl rollout status deployment/envoy-proxy -n ${NAMESPACE}
}

scale_kubernetes() {
    local replicas=${1:-3}
    log_info "Scaling PromptShield to ${replicas} replicas..."
    kubectl scale deployment/promptshield -n ${NAMESPACE} --replicas=${replicas}
    kubectl rollout status deployment/promptshield -n ${NAMESPACE}
}

monitor_deployment() {
    log_info "Monitoring deployment..."
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        docker compose -f docker-compose.production.yaml logs -f --tail=100
    elif [ "$DEPLOYMENT_TYPE" = "kubernetes" ]; then
        kubectl logs -f -n ${NAMESPACE} -l app=promptshield --tail=100
    fi
}

run_tests() {
    log_info "Running deployment tests..."
    
    # Test API endpoint
    API_URL="http://localhost:9090"
    if [ "$DEPLOYMENT_TYPE" = "kubernetes" ]; then
        API_URL=$(kubectl get ingress -n ${NAMESPACE} -o jsonpath='{.items[0].status.loadBalancer.ingress[0].hostname}')
    fi
    
    # Test health endpoint
    log_info "Testing health endpoint..."
    curl -sf ${API_URL}/healthz || log_error "Health check failed"
    
    # Test readiness endpoint
    log_info "Testing readiness endpoint..."
    curl -sf ${API_URL}/readyz || log_error "Readiness check failed"
    
    # Test prompt checking
    log_info "Testing prompt checking..."
    curl -X POST ${API_URL}/check \
        -H "Content-Type: text/plain" \
        -d "Hello world" \
        -w "\nHTTP Status: %{http_code}\n"
    
    log_info "✅ All tests passed"
}

# Main script
case "$1" in
    docker)
        check_prerequisites
        deploy_docker
        ;;
    kubernetes|k8s)
        check_prerequisites
        deploy_kubernetes
        ;;
    rollback)
        if [ "$2" = "docker" ]; then
            rollback_docker
        else
            rollback_kubernetes
        fi
        ;;
    scale)
        scale_kubernetes $2
        ;;
    monitor)
        monitor_deployment
        ;;
    test)
        run_tests
        ;;
    *)
        echo "PromptShield Deployment Script"
        echo ""
        echo "Usage: $0 {docker|kubernetes|rollback|scale|monitor|test} [environment]"
        echo ""
        echo "Commands:"
        echo "  docker       - Deploy using Docker Compose"
        echo "  kubernetes   - Deploy to Kubernetes cluster"
        echo "  rollback     - Rollback to previous version"
        echo "  scale N      - Scale to N replicas (Kubernetes only)"
        echo "  monitor      - Monitor deployment logs"
        echo "  test         - Run deployment tests"
        echo ""
        echo "Environments: production (default), staging, development"
        exit 1
        ;;
esac