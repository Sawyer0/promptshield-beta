#!/bin/bash
set -e

echo "=== PromptShield Kubernetes Deployment ==="
echo "Using Docker image: promptshield-v020-ps-enforcer:latest"

# Step 1: Apply the Kubernetes manifest
echo "Deploying to Kubernetes..."
kubectl apply -f deployments/kubernetes/enforcer.yaml

# Step 2: Wait for deployment
echo "Waiting for deployment to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/promptshield-enforcer -n promptshield || true

# Step 3: Check deployment status
echo "Checking deployment status..."
kubectl get pods -n promptshield
kubectl get svc -n promptshield

# Step 4: Set up port forwarding
echo "Setting up port forwarding..."
kubectl port-forward -n promptshield svc/promptshield-enforcer 9090:9090 9091:9091 &
PORT_FORWARD_PID=$!

sleep 3

# Step 5: Test health endpoint
echo "Testing health endpoint..."
curl -sf http://localhost:9090/healthz && echo "✓ Health check passed!" || echo "✗ Health check failed"

echo ""
echo "=== Deployment Complete ==="
echo "Services available at:"
echo "  HTTP: http://localhost:9090"
echo "  gRPC: localhost:9091"
echo ""
echo "Port forward PID: $PORT_FORWARD_PID"
echo "To stop port forwarding: kill $PORT_FORWARD_PID"