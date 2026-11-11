.PHONY: help build up down logs validate test run-curl run-go test-go clean

ENV_FILE := .env

help:
	@echo "Available commands:"
	@echo "  make build   	- Build or rebuild ONLY the Caddy proxy container"
	@echo "  make up      	- Auto-generate credentials if missing and start stack"
	@echo "  make down    	- Stop containers"
	@echo "  make logs    	- Stream unified JSON logs"
	@echo "  make validate - Build Caddy and validate its configuration"
	@echo "  make run-curl  - Run the proxy rate-limit smoke test"
	@echo "  make run-go  	- Run the Go client through the proxy"
	@echo "  make test-go 	- Run the Go retry-transport unit tests"
	@echo "  make clean   	- Stop containers and prune local artifacts"

build:
	docker compose build caddy

$(ENV_FILE):
	@echo "Generating fresh proxy credentials..."
	@USER_NAME="$${PROXY_USER:-my-custom-user}"; \
	PASS="$$(openssl rand -base64 16)"; \
	RAW_HASH="$$(docker run --rm caddy:latest caddy hash-password --plaintext "$$PASS")"; \
	printf "PROXY_USER=%s\nPROXY_PASS=%s\nPROXY_HASH=%s\nIE6=\n" "$$USER_NAME" "$$PASS" "$$RAW_HASH" > $(ENV_FILE)

up: $(ENV_FILE)
	docker compose up -d --wait

down:
	docker compose down

logs:
	@if [ -f ./tail-logs.sh ]; then \
		./tail-logs.sh; \
	else \
		docker compose logs -f; \
	fi

validate: $(ENV_FILE) build
	docker compose run --rm --no-deps caddy caddy validate --config /etc/caddy/Caddyfile

run-curl: up
	@./test_curl.sh

test: run-curl

run-go: up
	@cd cmd/app && \
		PROXY_USER="$$(sed -n 's/^PROXY_USER=//p' ../../.env)" \
		PROXY_PASS="$$(sed -n 's/^PROXY_PASS=//p' ../../.env)" \
		PROXY_URL="http://127.0.0.1:8090" \
		TARGET_URL="$${TARGET_URL:-http://www.yahoo.com}" \
		go run .

test-go:
	cd cmd/app && go test ./...

clean:
	docker compose down --rmi local --volumes --remove-orphans
	rm -f .env
