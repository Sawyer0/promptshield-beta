#!/usr/bin/env bash
set -euo pipefail

# E2E smoke test script for PromptShield CLI
# Usage: bash tools/e2e.sh

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

echo "Building..."
go build -ldflags "-X 'github.com/promptshield/promptshield/cmd.version=$(git describe --tags --always 2>/dev/null || echo dev)'" -o bin/promptshield ./cmd/promptshield

PS=./bin/promptshield

echo
echo "== Help =="
$PS || true

echo
echo "== Demo =="
$PS demo || true

echo
echo "== Stylish scan (color) =="
$PS scan:file demo/real-attacks.json --rulepack demo/rules.yaml || true

echo
echo "== JSON scan =="
$PS --output-format=json scan:file demo/real-attacks.json --rulepack demo/rules.yaml | head -n 5 || true

echo
echo "== GitHub annotations scan =="
$PS --output-format=github scan:file demo/real-attacks.json --rulepack demo/rules.yaml || true

echo
echo "== Quiet mode + exit code =="
set +e
$PS --quiet scan:file demo/real-attacks.json --rulepack demo/rules.yaml
echo "exit code: $?"
set -e

echo
echo "== Fail on ERROR (expect non-zero) =="
set +e
$PS scan:file --fail-on ERROR demo/real-attacks.json --rulepack demo/rules.yaml
echo "exit code: $?"
set -e

echo
echo "== Simulate CI defaults (JSON + fail on INFO) =="
set +e
CI=true $PS scan:file demo/real-attacks.json --rulepack demo/rules.yaml | head -n 5
echo "exit code: $?"
set -e

echo
echo "== Rules management =="
$PS rules:create --dest rules/example.yaml -f
$PS rules:list --path rules | head -n 10
$PS rules:validate --path rules || true

echo
echo "== Config print =="
$PS config print | head -n 20

echo
echo "E2E script completed."


