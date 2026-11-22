#!/bin/bash
set -euo pipefail

PROXY_URL="${PROXY_URL:-http://127.0.0.1:8090}"
PROXY_USER="${PROXY_USER:-test-proxy-user}"
PROXY_PASS="${PROXY_PASS:-test-proxy-password}"

echo "=== Testing health endpoint (expected HTTP 200) ==="
code=$(curl --silent --show-error -o /dev/null -w "%{http_code}" \
  --connect-timeout 5 \
  --max-time 20 \
  "$PROXY_URL/healthz")
echo "Request 1: HTTP status $code"
echo ""

echo "=== Testing missing credentials (expected HTTP 401) ==="
code=$(curl --silent --show-error -o /dev/null -w "%{http_code}" \
  --connect-timeout 5 \
  --max-time 20 \
  --header "Host: www.yahoo.com" \
  "$PROXY_URL/")
echo "Request 1: HTTP status $code"

echo "=== Testing yahoo.com (Target: 5 QPS cap - firing 7 rapid requests) ==="
for i in {1..7}; do
  code=$(curl --silent --show-error -o /dev/null -w "%{http_code}" \
    --connect-timeout 5 \
    --max-time 20 \
    --user "$PROXY_USER:$PROXY_PASS" \
    --header "Host: www.yahoo.com" \
    "$PROXY_URL/")
  echo "Request $i: HTTP status $code"
  sleep 0.05
done

echo "=== Testing sec.gov (Target: 3 QPS cap - firing 5 rapid requests) ==="
for i in {1..5}; do
  code=$(curl --silent --show-error -o /dev/null -w "%{http_code}" \
    --connect-timeout 5 \
    --max-time 20 \
    --user "$PROXY_USER:$PROXY_PASS" \
    --header "Host: www.sec.gov" \
    "$PROXY_URL/")
  echo "Request $i: HTTP status $code"
  sleep 0.05
done


echo ""
echo "=== Testing example.com (Not configured - expected HTTP 403) ==="
code=$(curl --silent --show-error -o /dev/null -w "%{http_code}" \
  --connect-timeout 5 \
  --max-time 20 \
  --user "$PROXY_USER:$PROXY_PASS" \
  --header "Host: www.example.com" \
  "$PROXY_URL/")
echo "Request 1: HTTP status $code"
