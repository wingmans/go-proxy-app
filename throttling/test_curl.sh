#!/usr/bin/env bash
set -euo pipefail

gateway_url=${PROXY_URL:-http://127.0.0.1:8090}

request_status() {
  local host=$1
  local path=${2:-/}
  local code

  code=$(curl --silent --show-error -o /dev/null -w '%{http_code}' \
    --connect-timeout 5 --max-time 20 \
    --header "Host: $host" "$gateway_url$path")
  printf '%s' "$code"
}

expect_status() {
  local want=$1
  local host=$2
  local path=${3:-/}
  local got

  got=$(request_status "$host" "$path")
  if [[ $got != "$want" ]]; then
    echo "request for $host$path returned $got, want $want" >&2
    exit 1
  fi
}

expect_success() {
  local host=$1
  local got

  got=$(request_status "$host")
  if [[ $got != 2* && $got != 3* ]]; then
    echo "request for $host returned $got, want 2xx or 3xx" >&2
    exit 1
  fi
}

health=$(curl --silent --show-error -o /dev/null -w '%{http_code}' \
  --connect-timeout 2 --max-time 5 "$gateway_url/healthz")
[[ $health == 200 ]] || {
  echo "health endpoint returned $health, want 200" >&2
  exit 1
}

expect_success example.com
sleep 1.1

# The third request is rejected even though the first two use different hosts.
expect_success example.com
expect_success www.example.com
expect_status 429 example.com

sleep 1.1
expect_status 403 example.org

echo 'gateway smoke test passed'
