FROM caddy:2.11.9-builder-alpine AS builder

ENV RATELIMIT_MODULE=github.com/mholt/caddy-ratelimit \
    RATELIMIT_VERSION=v0.1.1-0.20260612195517-5625512f24f6

COPY retry-after-ms.patch /tmp/retry-after-ms.patch
RUN go mod download "$RATELIMIT_MODULE@$RATELIMIT_VERSION" \
    && cp -R "$(go env GOMODCACHE)/$RATELIMIT_MODULE@$RATELIMIT_VERSION" /tmp/caddy-ratelimit \
    && chmod -R u+w /tmp/caddy-ratelimit \
    && git -C /tmp/caddy-ratelimit apply /tmp/retry-after-ms.patch \
    && xcaddy build --with "$RATELIMIT_MODULE@$RATELIMIT_VERSION=/tmp/caddy-ratelimit"

FROM caddy:2.11.9-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
