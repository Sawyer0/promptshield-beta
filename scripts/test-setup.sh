#!/bin/bash

# Test script for PromptShield Docker setup
set -e

echo "🧪 Testing PromptShield setup..."

# Test enforcer health
echo "Testing enforcer health..."
if curl -s http://localhost:9090/healthz | grep -q "ok"; then
    echo "✅ Enforcer is healthy"
else
    echo "❌ Enforcer health check failed"
    exit 1
fi

# Test frontend BFF health
echo "Testing frontend BFF health..."
if curl -s http://localhost:3001/api/healthz | grep -q "ok"; then
    echo "✅ Frontend BFF is healthy"
else
    echo "❌ Frontend BFF health check failed"
    exit 1
fi

# Test nginx proxy
echo "Testing nginx proxy..."
if curl -s http://localhost:80/healthz | grep -q "ok"; then
    echo "✅ Nginx proxy is working"
else
    echo "❌ Nginx proxy health check failed"
    exit 1
fi

# Test a simple enforcement request
echo "Testing enforcement endpoint..."
response=$(curl -s -X POST http://localhost:9090/check \
  -H 'content-type: text/plain' \
  --data 'Hello, how can I help you today?')

if echo "$response" | grep -q "allow"; then
    echo "✅ Enforcement endpoint is working"
else
    echo "❌ Enforcement endpoint test failed"
    echo "Response: $response"
    exit 1
fi

# Test frontend API proxy
echo "Testing frontend API proxy..."
if curl -s http://localhost:3001/api/proxy/healthz | grep -q "ok"; then
    echo "✅ Frontend API proxy is working"
else
    echo "❌ Frontend API proxy test failed"
    exit 1
fi

echo ""
echo "🎉 All tests passed! Your PromptShield setup is working correctly."
echo ""
echo "📱 You can now access:"
echo "   Frontend UI: http://localhost:3001"
echo "   Enforcer API: http://localhost:9090"
echo "   Nginx Proxy: http://localhost:80"
