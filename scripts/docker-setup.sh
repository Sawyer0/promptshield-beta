#!/bin/bash

# Docker setup script for PromptShield
set -e

echo "🚀 Setting up PromptShield Docker stack..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker Desktop and try again."
    exit 1
fi

# Build and start the stack
echo "📦 Building and starting containers..."
docker compose up -d --build

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check service health
echo "🔍 Checking service health..."

# Check enforcer health
if curl -s http://localhost:9090/healthz > /dev/null; then
    echo "✅ Enforcer health check passed"
else
    echo "❌ Enforcer health check failed"
fi

# Check frontend BFF health
if curl -s http://localhost:3001/api/healthz > /dev/null; then
    echo "✅ Frontend BFF health check passed"
else
    echo "❌ Frontend BFF health check failed"
fi

# Check nginx proxy
if curl -s http://localhost:80/healthz > /dev/null; then
    echo "✅ Nginx proxy health check passed"
else
    echo "❌ Nginx proxy health check failed"
fi

echo ""
echo "🎉 Setup complete!"
echo ""
echo "📱 Access points:"
echo "   Frontend UI: http://localhost:3001"
echo "   Enforcer API: http://localhost:9090"
echo "   Nginx Proxy: http://localhost:80"
echo "   Enforcement Proxy: http://localhost:8080"
echo "   Grafana: http://localhost:3000 (admin/admin)"
echo "   Prometheus: http://localhost:9092"
echo ""
echo "🔧 Useful commands:"
echo "   View logs: docker compose logs -f"
echo "   Stop stack: docker compose down"
echo "   Restart: docker compose restart"
echo "   Rebuild: docker compose up -d --build"
