# Phase 1: Build the custom Caddy binary with plugins
FROM caddy:builder AS builder

RUN xcaddy build \
    --with github.com/mholt/caddy-ratelimit \
    --with github.com/caddyserver/forwardproxy

# Phase 2: Create the lean production runtime image
FROM caddy:latest

COPY --from=builder /usr/bin/caddy /usr/bin/caddy