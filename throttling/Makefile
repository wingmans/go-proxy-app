.PHONY: help build configure up down status logs validate test run-curl \
	run-go test-go test-generator clean require-rules

ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
RULES_FILE ?= $(ROOT_DIR)/policies/example.rules
CONFIG_NAME ?= default
GENERATED_DIR := $(ROOT_DIR)/.generated
CADDYFILE := $(GENERATED_DIR)/$(CONFIG_NAME).Caddyfile
PROXY_IMAGE ?= go-proxy-app-throttling:0.1.0
PROXY_PORT ?= 8090
TEST_PORT ?= 18090
TEST_PROJECT ?= go-proxy-app-throttling-test
TEST_RULES_FILE := $(ROOT_DIR)/policies/test.rules

help:
	@printf '%s\n' \
		'Throttling gateway commands:' \
		'' \
		'  make build                 Build the versioned local image' \
		'  make up                   Start with policies/example.rules' \
		'  make up RULES_FILE=<path> Override the default policy' \
		'  make down                  Stop the gateway' \
		'  make status                Show container status' \
		'  make logs                  Follow gateway logs' \
		'  make validate             Validate the default policy' \
		'  make validate RULES_FILE=<path>' \
		'                             Validate generated Caddy configuration' \
		'  make test                  Run generator and gateway tests' \
		'  make run-go               Run with the default policy' \
		'  make run-go RULES_FILE=<path>' \
		'                             Run the retry-client example' \
		'  make clean                 Stop containers and generated files'

build:
	PROXY_IMAGE="$(PROXY_IMAGE)" docker compose build caddy

require-rules:
	@test -n "$(RULES_FILE)" || { \
		echo 'RULES_FILE is required' >&2; exit 1; \
	}
	@test -f "$(RULES_FILE)" || { \
		echo 'RULES_FILE does not exist: $(RULES_FILE)' >&2; exit 1; \
	}

configure: require-rules
	@mkdir -p "$(GENERATED_DIR)"
	cd cmd/generate && go run . \
		-rules "$(abspath $(RULES_FILE))" -out "$(CADDYFILE)"

up: configure
	@POLICY_HASH="$$(shasum -a 256 "$(RULES_FILE)" | awk '{print $$1}')"; \
	CADDYFILE_PATH="$(CADDYFILE)" \
	POLICY_HASH="$$POLICY_HASH" \
	PROXY_IMAGE="$(PROXY_IMAGE)" \
	PROXY_PORT="$(PROXY_PORT)" \
		docker compose up -d --wait caddy

down:
	@CADDYFILE_PATH="$(CADDYFILE)" \
	PROXY_IMAGE="$(PROXY_IMAGE)" \
	PROXY_PORT="$(PROXY_PORT)" \
		docker compose down

status:
	@CADDYFILE_PATH="$(CADDYFILE)" \
	PROXY_IMAGE="$(PROXY_IMAGE)" \
	PROXY_PORT="$(PROXY_PORT)" \
		docker compose ps

logs:
	@CADDYFILE_PATH="$(CADDYFILE)" \
	PROXY_IMAGE="$(PROXY_IMAGE)" \
	PROXY_PORT="$(PROXY_PORT)" \
		docker compose logs -f caddy

validate: configure build
	@CADDYFILE_PATH="$(CADDYFILE)" \
	PROXY_IMAGE="$(PROXY_IMAGE)" \
	PROXY_PORT="$(PROXY_PORT)" \
		docker compose run --rm --no-deps caddy \
		caddy validate --config /etc/caddy/Caddyfile

test-generator:
	cd cmd/generate && go test ./...

test-go:
	cd cmd/app && go test ./...

run-curl:
	@set -eu; \
	cleanup() { \
		$(MAKE) --no-print-directory down \
			CONFIG_NAME=test PROXY_PORT="$(TEST_PORT)" \
			PROXY_IMAGE="$(PROXY_IMAGE)" \
			COMPOSE_PROJECT_NAME="$(TEST_PROJECT)" >/dev/null; \
	}; \
	trap cleanup EXIT INT TERM; \
	$(MAKE) --no-print-directory up \
		RULES_FILE="$(TEST_RULES_FILE)" CONFIG_NAME=test \
		PROXY_PORT="$(TEST_PORT)" PROXY_IMAGE="$(PROXY_IMAGE)" \
		COMPOSE_PROJECT_NAME="$(TEST_PROJECT)"; \
	published="$$(CADDYFILE_PATH="$(GENERATED_DIR)/test.Caddyfile" \
		PROXY_PORT="$(TEST_PORT)" PROXY_IMAGE="$(PROXY_IMAGE)" \
		COMPOSE_PROJECT_NAME="$(TEST_PROJECT)" \
		docker compose port caddy 8080)"; \
	case "$$published" in \
		127.0.0.1:*) ;; \
		*) echo "gateway is not loopback-only: $$published" >&2; exit 1 ;; \
	esac; \
	PROXY_URL="http://127.0.0.1:$(TEST_PORT)" ./test_curl.sh; \
	cleanup; \
	trap - EXIT INT TERM; \
	if curl --silent --max-time 1 \
		"http://127.0.0.1:$(TEST_PORT)/healthz" >/dev/null 2>&1; then \
		echo 'gateway remained reachable after shutdown' >&2; exit 1; \
	fi

test: test-generator test-go run-curl

run-go: up
	@cd cmd/app && \
		GATEWAY_URL="http://127.0.0.1:$(PROXY_PORT)" \
		TARGET_URL="$${TARGET_URL:-https://example.com}" \
		go run .

clean:
	@CADDYFILE_PATH="$(CADDYFILE)" \
	PROXY_IMAGE="$(PROXY_IMAGE)" \
	PROXY_PORT="$(PROXY_PORT)" \
		docker compose down --remove-orphans
	rm -rf "$(GENERATED_DIR)"
