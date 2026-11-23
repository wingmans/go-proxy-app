SHELL := /bin/bash

.PHONY: help env create up down logs validate run-curl run-go test test-go

ENV_FILE := .env
ENV_SAMPLE := .env.sample

help:
	@echo "Available commands:"
	@echo "  make create    Create the authorization proxy container"
	@echo "  make up        Start the authorization proxy"
	@echo "  make down      Stop the authorization proxy"
	@echo "  make logs      Follow proxy logs"
	@echo "  make validate  Validate the Caddy configuration"
	@echo "  make run-curl  Curl the already-running proxy"
	@echo "  make run-go    Test the proxy from the host with Go"
	@echo "  make test      Run Go tests and curl the running proxy"
	@echo "  make test-go   Run the Go unit tests"

env:
	@[[ -f "$(ENV_FILE)" ]] \
		&& grep -q '^API_UPSTREAM=' "$(ENV_FILE)" \
		&& grep -q '^API_AUTH_HEADER=' "$(ENV_FILE)" \
		&& grep -q '^API_AUTH_VALUE=' "$(ENV_FILE)" \
		&& grep -q '^AUTH_PROXY_USER=' "$(ENV_FILE)" \
		&& grep -q '^AUTH_PROXY_PASS=' "$(ENV_FILE)" \
		&& grep -q '^AUTH_PROXY_HASH=' "$(ENV_FILE)" \
		&& grep -q '^AUTH_PROXY_PORT=' "$(ENV_FILE)" \
		|| { cp "$(ENV_SAMPLE)" "$(ENV_FILE)"; echo "Created $(ENV_FILE) from $(ENV_SAMPLE)."; }

create: env
	docker compose create --build

up: env
	docker compose up -d --build --wait

down:
	docker compose down

logs:
	docker compose logs -f caddy

validate: env
	docker compose run --rm --no-deps caddy caddy validate --config /etc/caddy/Caddyfile

run-curl:
	./test.sh

run-go:
	go run ./cmd/client

test: test-go run-curl

test-go:
	go test ./...
