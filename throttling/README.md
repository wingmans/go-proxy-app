# Throttling gateway

This directory builds a local HTTP gateway that rate-limits requests before
forwarding them to allowlisted HTTPS services.

It is deliberately not a standard HTTPS forward proxy. A forward proxy sees
only the initial `CONNECT` for a reused TLS tunnel and therefore cannot enforce
a request-per-second limit on the encrypted requests inside that tunnel. This
gateway terminates the local HTTP request, applies its policy, and creates the
HTTPS request to the upstream service itself.

The gateway listens on `http://127.0.0.1:8090` by default. It has no
authentication because Docker publishes it only on the host loopback
interface.

## Policy

The gateway has no built-in destinations. A policy file is required at
startup. Each non-comment line contains a bucket name, a rate, and one or more
allowed upstream hosts:

    sec        8r/s  sec.gov  *.sec.gov
    yahoo      5r/s  query1.finance.yahoo.com
    ecb        5r/s  data-api.ecb.europa.eu
    wikipedia  5r/s  en.wikipedia.org

Rates use `r/s`, `r/m`, or `r/h`. Bucket names must start with a lowercase
letter and may contain lowercase letters, digits, and underscores. Hosts may
be exact names or start with `*.`.

All hosts on one line share one sliding-window budget. In the example,
`sec.gov`, `www.sec.gov`, and `data.sec.gov` consume the same eight-request
budget. An excess request receives `429 Too Many Requests`. A host absent from
the policy receives `403 Forbidden`.

The generator rejects empty policies, malformed rates or hosts, duplicate
buckets, and hosts assigned to multiple buckets. There is no unrestricted
fallback.

The repository contains a neutral policy at
[`policies/example.rules`](policies/example.rules). Real applications should
own their policy files and pass their paths to this repository.

## Run it

Build the pinned, versioned local image:

    make build

Start the gateway with the default `policies/example.rules` policy:

    cp .env.example .env
    make up

Docker Compose and `test_curl.sh` both load `.env`. The checked-in example
contains public local-test credentials. Edit `.env` to change the credentials
or port without changing the Makefile or scripts.

Override the policy when needed:

    make up RULES_FILE=policies/example.rules

`make up` generates the Caddy configuration, builds the image if necessary,
and waits for the running gateway healthcheck. Repeating the command is safe.
Changing the policy recreates the container but does not rebuild the image.

Inspect or stop it with:

    make status
    make logs
    make down

Validate the default generated configuration without starting the long-running
container:

    make validate

Or validate a different policy:

    make validate RULES_FILE=policies/example.rules

## Client contract

Clients connect to the local gateway over HTTP while preserving the intended
HTTPS upstream in the request's `Host` header. The method, path, query, body,
and application headers remain unchanged.

For example, this local request asks the gateway to fetch
`https://example.com/`:

    curl -H 'Host: example.com' http://127.0.0.1:8090/

The sample in `cmd/app` contains a concrete Go `http.RoundTripper` that
performs this destination rewrite. It rejects non-HTTPS upstream URLs. Run it
against a host allowed by the selected policy:

    make run-go RULES_FILE=policies/example.rules

If the gateway is stopped, the client receives a connection error. The client
must never fall back to its original external destination.

## Test it

    make test

The test target:

- tests policy parsing and Caddyfile generation;
- tests request rewriting and `429` retry behavior;
- starts an isolated gateway on `127.0.0.1:18090`;
- verifies allowed requests, a shared rate bucket, `429`, and `403`;
- verifies the Docker port is loopback-only;
- stops the test gateway and verifies that it is no longer reachable.

The smoke test uses the reserved `example.com` service. It does not send rate
test traffic to SEC or another application endpoint.

## Add a destination

Add the host to the application-owned policy under an existing bucket or a
new bucket, then run `make up` again with that policy. Do not add permissive
catch-all rules.

## Versioning

The Dockerfile pins Caddy and the rate-limit module. The default local image is
named `go-proxy-app-throttling:0.1.0`; override `PROXY_IMAGE` when needed.

Publishing the image to a registry is optional. A source clone plus
`make build` provides the same runtime behavior and is the normal workflow.
