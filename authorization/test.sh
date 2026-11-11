#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_NAME=authorization-test

cleanup() {
	docker compose \
		-p "$PROJECT_NAME" \
		-f "$SCRIPT_DIR/docker-compose.yml" \
		-f "$SCRIPT_DIR/compose.test.yml" \
		down --volumes --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

# Values are required while Compose interpolates the base file. The test
# override replaces them inside the container with its mock credentials.
export API_UPSTREAM=http://mock-api:8080
export API_AUTH_HEADER=X-Demo-Api-Key
export API_AUTH_VALUE=test-secret
export AUTH_PROXY_PORT=18091

cleanup
docker compose \
	-p "$PROJECT_NAME" \
	-f "$SCRIPT_DIR/docker-compose.yml" \
	-f "$SCRIPT_DIR/compose.test.yml" \
	up -d --build --wait

run_curl() {
	response=$(curl --silent --show-error --fail \
		-H 'Authorization: Bearer caller-secret-must-not-pass-through' \
		-H 'Cookie: caller-cookie-must-not-pass-through' \
		-H 'X-Demo-Api-Key: wrong-value' \
		http://127.0.0.1:18091/v1/example)

	if [ "$response" != '{"authenticated":true,"path":"/v1/example"}' ]; then
		echo "unexpected curl response: $response" >&2
		exit 1
	fi

	echo "curl authorization test passed"
}

run_go() {
	(
		cd "$SCRIPT_DIR"
		AUTH_PROXY_URL=http://127.0.0.1:18091 go run ./cmd/client
	)

	echo "Go authorization client test passed"
}

case "${1:-all}" in
	curl)
		run_curl
		;;
	go)
		run_go
		;;
	all)
		run_curl
		run_go
		;;
	*)
		echo "usage: $0 [curl|go|all]" >&2
		exit 2
		;;
esac

echo "authorization proxy integration test passed"
