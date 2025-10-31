#!/bin/bash
set -e

echo "========================================="
echo "   PromptShield Full Stack Deployment"
echo "========================================="
echo ""

# Step 1: Deploy to Kubernetes
echo "Step 1: Deploying to Kubernetes..."
echo "-------------------------------------"
./k8s-deploy.sh
echo ""

# Give services time to stabilize
echo "Waiting for services to stabilize..."
sleep 10

# Step 2: Set up ngrok tunnel
echo "Step 2: Setting up ngrok tunnel..."
echo "-------------------------------------"
./ngrok-setup.sh &
NGROK_SETUP_PID=$!

# Wait for ngrok to be ready
sleep 10

# Get ngrok URL
NGROK_URL=$(curl -s http://localhost:4040/api/tunnels 2>/dev/null | grep -o '"public_url":"[^"]*' | grep -o 'https://[^"]*' | head -1)

if [ -z "$NGROK_URL" ]; then
    echo "Warning: Could not get ngrok URL, using localhost"
    NGROK_URL="http://localhost:9090"
fi

echo ""
echo "Step 3: Running integration tests..."
echo "-------------------------------------"
./test-integration.sh "$NGROK_URL"

echo ""
echo "========================================="
echo "   Deployment Complete!"
echo "========================================="
echo ""
echo "Access Points:"
echo "  Local HTTP:  http://localhost:9090"
echo "  Local gRPC:  localhost:9091"
echo "  Public URL:  $NGROK_URL"
echo ""
echo "Dashboards:"
echo "  ngrok:       http://localhost:4040"
echo "  Metrics:     http://localhost:9090/metrics"
echo ""
echo "Quick test commands:"
echo "  curl $NGROK_URL/healthz"
echo "  curl -X POST $NGROK_URL/check -H 'Content-Type: text/plain' -d 'Hello world'"
echo ""
echo "To view logs:"
echo "  kubectl logs -f -n promptshield -l app=promptshield-enforcer"
echo ""
echo "To stop all services:"
echo "  kubectl delete -f deployments/kubernetes/enforcer.yaml"
echo "  pkill -f 'kubectl port-forward'"
echo "  pkill -f ngrok"