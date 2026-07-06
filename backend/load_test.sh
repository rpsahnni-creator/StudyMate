#!/usr/bin/env bash
# Load test using hey (https://github.com/rakyll/hey)
# Run: bash load_test.sh [BASE_URL] [JWT_TOKEN]
#
# Install hey:
#   go install github.com/rakyll/hey@latest

set -euo pipefail

BASE_URL=${1:-http://localhost:8080}
TOKEN=${2:-your-test-jwt-token}

if ! command -v hey >/dev/null 2>&1; then
  echo "hey is not installed. Run: go install github.com/rakyll/hey@latest"
  exit 1
fi

echo "=== Testing /health ==="
hey -n 1000 -c 50 "${BASE_URL}/health"

echo ""
echo "=== Testing /ready ==="
hey -n 500 -c 20 "${BASE_URL}/ready"

echo ""
echo "=== Testing /me/features ==="
hey -n 500 -c 20 -H "Authorization: Bearer ${TOKEN}" \
  "${BASE_URL}/me/features"

echo ""
echo "=== Testing /users/me/reports ==="
hey -n 200 -c 10 -H "Authorization: Bearer ${TOKEN}" \
  "${BASE_URL}/users/me/reports"

echo ""
echo "Load test complete. Check Grafana for metrics at ${BASE_URL}/metrics"
