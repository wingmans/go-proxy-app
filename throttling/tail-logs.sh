#!/usr/bin/env bash
set -euo pipefail

# Ensure jq is available
if ! command -v jq &> /dev/null; then
    echo "Error: 'jq' is required for JSON log formatting. Install it via your package manager." >&2
    exit 1
fi

echo "=================================================================="
echo " Streaming unified logs from Caddy Proxy and Go Application..."
echo " Press Ctrl+C to exit."
echo "=================================================================="

# Tail docker compose logs, parse container name and JSON payload
docker compose logs -f --tail=20 2>&1 | while IFS= read -r line; do
    # Docker compose log format: "container_name  | {json...}"
    if [[ "$line" =~ ^([^[:space:]]+)[[:space:]]+\|[[:space:]](.*)$ ]]; then
        container="${BASH_REMATCH[1]}"
        json_payload="${BASH_REMATCH[2]}"

        # Process using jq to normalize output for both services
        echo "$json_payload" | jq -c \
            --arg container "$container" \
            '
            if $container == "caddy-proxy" then
                # Format Caddy logs: Highlight status codes and requested hosts
                {
                    src: $container,
                    status: .status,
                    method: .request.method,
                    host: .request.host,
                    uri: .request.uri
                }
            else
                # Format Go App slog logs: Highlight backoffs, attempts, and messages
                {
                    src: $container,
                    level: .level,
                    msg: .msg,
                    host: .host,
                    attempt: .attempt,
                    backoff: .backoff_delay
                }
            end
            ' 2>/dev/null || echo "[$container] $json_payload"
    else
        echo "$line"
    fi
done
