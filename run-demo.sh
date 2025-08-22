#!/usr/bin/env bash
set -euo pipefail

# Wrapper script to run demos with environment loaded
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Load environment from .env file if it exists
if [[ -f ".env" ]]; then
    echo "Loading environment from .env file..."
    set -a  # automatically export all variables
    source ".env"
    set +a
    echo "Loaded OPENAI_API_KEY: ${OPENAI_API_KEY:+***set***}"
else
    echo "Warning: No .env file found"
fi

# Export demo variables
export GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
export PROVIDER_URL="${PROVIDER_URL:-https://api.openai.com}"
export DEMO_USE_MOCKS="${DEMO_USE_MOCKS:-0}"
export MODEL_NAME="${MODEL_NAME:-gpt-4o-mini}"

# Ensure all variables are exported
export OPENAI_API_KEY="${OPENAI_API_KEY:-}"
export ADMIN_TOKEN="${ADMIN_TOKEN:-}"

echo "Environment configured for demo"
echo "Running core demo..."

# Run the demo with relaxed error handling
set +e
./tools/demos/core-demo.sh
demo_exit_code=$?
set -e

if [ $demo_exit_code -ne 0 ]; then
    echo "Demo completed with exit code: $demo_exit_code"
fi