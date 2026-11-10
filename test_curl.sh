#!/bin/bash

# Load credentials dynamically from .env
if [ -f .env ]; then
  export PROXY_USER=$(grep -E '^PROXY_USER=' .env | cut -d'=' -f2-)
  export PROXY_PASS=$(grep -E '^PROXY_PASS=' .env | cut -d'=' -f2-)
fi

if [ -z "$PROXY_USER" ] || [ -z "$PROXY_PASS" ]; then
  echo "Error: PROXY_USER or PROXY_PASS not found in .env"
  exit 1
fi

PROXY_URL="http://${PROXY_USER}:${PROXY_PASS}@127.0.0.1:8090"



echo "=== Testing yahoo.com (Target: 2 QPS cap - firing 4 rapid requests) ==="
for i in {1..4}; do
  code=$(curl -s -o /dev/null -w "%{http_code}" \
    --proxy "$PROXY_URL" \
    http://www.yahoo.com)
  echo "Request $i: HTTP status $code"
  sleep 0.05
done

echo "=== Testing sec.gov (Target: 10 QPS cap) ==="
for i in {1..15}; do
  code=$(curl -s -o /dev/null -w "%{http_code}" \
    --proxy "$PROXY_URL" \
    http://www.sec.gov)
  echo "Request $i: HTTP status $code"
  sleep 0.05
done


echo ""
echo "=== Testing example.com (Unrestricted - firing 5 rapid requests) ==="
for i in {1..5}; do
  code=$(curl -s -o /dev/null -w "%{http_code}" \
    --proxy "$PROXY_URL" \
    http://www.example.com)
  echo "Request $i: HTTP status $code"
  sleep 0.05
done
