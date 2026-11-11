.PHONY: help create up down logs validate run-curl run-go test test-go

ENV_FILE := .env

help:
	@echo "Available commands:"
	@echo "  make create    Create the authorization proxy container"
	@echo "  make up        Start the authorization proxy"
	@echo "  make down      Stop the authorization proxy"
	@echo "  make logs      Follow proxy logs"
	@echo "  make validate  Validate the Caddy configuration"
	@echo "  make run-curl  Test the proxy from the host with curl"
	@echo "  make run-go    Test the proxy from the host with Go"
	@echo "  make test      Run both proxy integration clients"
	@echo "  make test-go   Run the Go unit tests"

$(ENV_FILE): .env.example
	cp .env.example $(ENV_FILE)
	@echo "Created $(ENV_FILE); replace the example API credential before real use."

create: $(ENV_FILE)
	docker compose create

up: $(ENV_FILE)
	docker compose up -d --wait

down:
	docker compose down

logs:
	docker compose logs -f caddy

validate: $(ENV_FILE)
	docker compose run --rm --no-deps caddy caddy validate --config /etc/caddy/Caddyfile

run-curl:
	./test.sh curl

run-go:
	./test.sh go

test:
	./test.sh all

test-go:
	go test ./...
