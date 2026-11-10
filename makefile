.PHONY: help build up down logs test clean

help:
	@echo "Available commands:"
	@echo "  make build   - Build or rebuild ONLY the Caddy proxy container"
	@echo "  make up      - Auto-generate credentials if missing and start stack"
	@echo "  make down    - Stop containers"
	@echo "  make logs    - Stream unified JSON logs"
	@echo "  make test    - Run Go unit tests in ephemeral container (Go 1.27.3)"
	@echo "  make clean   - Stop containers and prune local artifacts"

build:
	docker compose build caddy

up:
	@if [ ! -f .env ]; then \
		echo "Generating fresh proxy credentials..."; \
		USER_NAME="$${PROXY_USER:-my-custom-user}"; \
		PASS="$$(openssl rand -base64 16)"; \
		RAW_HASH="$$(docker run --rm caddy:latest caddy hash-password --plaintext "$$PASS")"; \
		printf "PROXY_USER=%s\nPROXY_PASS=%s\nPROXY_HASH=%s\nIE6=\n" "$$USER_NAME" "$$PASS" "$$RAW_HASH" > .env; \
	fi
	docker compose up -d

down:
	docker compose down

logs:
	@if [ -f ./tail-logs.sh ]; then \
		./tail-logs.sh; \
	else \
		docker compose logs -f; \
	fi

test:
	@./test_curl.sh

clean:
	docker compose down --rmi local --volumes --remove-orphans
	rm -f .env