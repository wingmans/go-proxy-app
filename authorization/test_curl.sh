#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

set -a
source "$SCRIPT_DIR/.env"
set +a

AUTH_PROXY_PORT=${AUTH_PROXY_PORT:-8091}
AUTH_PROXY_URL=${AUTH_PROXY_URL:-http://127.0.0.1:$AUTH_PROXY_PORT}
AUTH_PROXY_USER=${AUTH_PROXY_USER:-test-proxy-user}
AUTH_PROXY_PASS=${AUTH_PROXY_PASS:-test-proxy-password}

echo "=== Health endpoint (expected HTTP 200) ==="
curl --silent --show-error \
	--output /dev/null \
	--write-out 'HTTP %{http_code}\n' \
	"$AUTH_PROXY_URL/healthz"

echo
echo "=== Missing proxy credentials (expected HTTP 401) ==="
curl --silent --show-error \
	--output /dev/null \
	--write-out 'HTTP %{http_code}\n' \
	"$AUTH_PROXY_URL/v1/example"

echo
echo "=== Basic credential replacement and path forwarding (expected HTTP 200) ==="
curl --silent --show-error \
	--user "$AUTH_PROXY_USER:$AUTH_PROXY_PASS" \
	--header 'Cookie: caller-cookie-must-not-pass-through' \
	--write-out '\nHTTP %{http_code}\n' \
	"$AUTH_PROXY_URL/v1/example"
