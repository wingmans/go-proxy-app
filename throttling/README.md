# Throttling proxy

This experiment builds Caddy with the forward-proxy and rate-limit modules. It
provides an authenticated HTTP forward proxy and applies host-specific request
limits:

- `*.sec.gov`: 10 requests per second
- `*.yahoo.com`: 2 requests per second
- other hosts: unrestricted

## Run it

```sh
make up
make test
make run-go
```

`make up` creates `.env` with a random proxy password on first use. See
`Caddyfile` to change hosts or limits, and run `make logs` to follow Caddy's
JSON access logs.

`make run-go` runs the Go retry client through the proxy against Yahoo by
default. Override the destination when needed:

```sh
TARGET_URL=http://www.example.com make run-go
```

Stop and remove the stack with:

```sh
make clean
```

The `cmd/app` directory contains a Go client experiment that retries responses
with status `429 Too Many Requests` using exponential backoff and jitter.
