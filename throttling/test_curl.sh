#!/bin/bash
set -euo pipefail

# Load credentials dynamically from .env
if [ -f .env ]; then
  export PROXY_USER=$(grep -E '^PROXY_USER=' .env | cut -d'=' -f2-)
  export PROXY_PASS=$(grep -E '^PROXY_PASS=' .env | cut -d'=' -f2-)
fi

if [ -z "${PROXY_USER:-}" ] || [ -z "${PROXY_PASS:-}" ]; then
  echo "Error: PROXY_USER or PROXY_PASS not found in .env"
  exit 1
fi

PROXY_URL="http://127.0.0.1:8090"

request_status() {
  local target_url=$1
  local code

  if ! code=$(curl --silent --show-error -o /dev/null -w "%{http_code}" \
    --connect-timeout 5 \
    --max-time 20 \
    --proxy "$PROXY_URL" \
    --proxy-user "$PROXY_USER:$PROXY_PASS" \
    "$target_url"); then
    echo "Proxy request failed. Check 'docker compose ps' and 'docker compose logs caddy'." >&2
    return 1
  fi

  printf '%s' "$code"
}

echo "=== Testing yahoo.com (Target: 2 QPS cap - firing 4 rapid requests) ==="
for i in {1..4}; do
  code=$(request_status http://www.yahoo.com)
  echo "Request $i: HTTP status $code"
  sleep 0.05
done

echo "=== Testing sec.gov (Target: 10 QPS cap) ==="
for i in {1..15}; do
  code=$(request_status http://www.sec.gov)
  echo "Request $i: HTTP status $code"
  sleep 0.05
done


echo ""
echo "=== Testing example.com (Unrestricted - firing 20 rapid requests) ==="
for i in {1..20}; do
  code=$(request_status http://www.example.com)
  echo "Request $i: HTTP status $code"
  sleep 0.05
done
