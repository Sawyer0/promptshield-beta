#!/usr/bin/env bash
set -euo pipefail

# Setup environment for PromptShield demos and development
# Usage: source scripts/setup-env.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Load environment from .env file if it exists
if [[ -f "$PROJECT_ROOT/.env" ]]; then
    echo "Loading environment from .env file..."
    set -a  # automatically export all variables
    source "$PROJECT_ROOT/.env"
    set +a
else
    echo "Warning: No .env file found at $PROJECT_ROOT/.env"
    echo "Create one with your API keys and configuration"
fi

# Export common demo variables
export GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
export PROVIDER_URL="${PROVIDER_URL:-https://api.openai.com}"
export DEMO_USE_MOCKS="${DEMO_USE_MOCKS:-0}"
export MODEL_NAME="${MODEL_NAME:-gpt-4o-mini}"

# Verify required variables are set
if [[ -z "${OPENAI_API_KEY:-}" ]]; then
    echo "Warning: OPENAI_API_KEY not set"
fi

if [[ -z "${ADMIN_TOKEN:-}" ]]; then
    echo "Info: ADMIN_TOKEN not set (optional for demos)"
fi

echo "Environment configured:"
echo "  GATEWAY_URL: $GATEWAY_URL"
echo "  PROVIDER_URL: $PROVIDER_URL" 
echo "  MODEL_NAME: $MODEL_NAME"
echo "  DEMO_USE_MOCKS: $DEMO_USE_MOCKS"
echo "  OPENAI_API_KEY: ${OPENAI_API_KEY:+***set***}"
echo "  ADMIN_TOKEN: ${ADMIN_TOKEN:+***set***}"