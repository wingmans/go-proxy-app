# Throttling Reverse Proxy Plan

## Summary

Replace the current Caddy forward-proxy design with a request-visible reverse proxy. This is required because a standard HTTPS proxy only sees the initial `CONNECT` request and cannot count individual requests inside a reused TLS tunnel.

Callers will rewrite upstream URLs:

```text
https://data.sec.gov/submissions/example.json
```

to:

```text
http://proxy-host:8090/proxy/data.sec.gov/submissions/example.json
```

Caddy will forward the request over HTTPS, enforce a domain-based requests-per-second limit, and return HTTP `429` locally when the limit is exceeded.

The implementation must remain small and understandable.

## Documented decisions

| Topic | Decision | Reason |
|---|---|---|
| Proxy type | Request-visible reverse proxy | Reliably counts every HTTPS request |
| Client integration | Rewrite requests to the proxy base URL | The calling code is controlled and can perform the rewrite |
| URL format | `/proxy/<target-host>/<original-path>` | Explicit, supports multiple domains and leaves room for `/healthz` |
| Rule dimension | Domain pattern | URL paths do not have separate limits |
| Wildcard behavior | Apex and all subdomains share one bucket | `*.sec.gov` counts `sec.gov`, `www.sec.gov`, and `data.sec.gov` together |
| Rule source | `rules.txt` | Simple, readable configuration |
| Generation | `make up` generates the Caddyfile first | Rule changes are applied without a separate manual step |
| Limit behavior | Reject excess requests with `429` | The proxy does not queue, delay, or retry requests |
| Unknown domains | Reject with a clear `403` | Traffic without an explicit rule must never pass |
| Authentication | Keep HTTP Basic authentication | Reuses the existing credential workflow |
| Listener | Port `8090`, reachable on all interfaces | Supports local and later cloud deployment |
| Demonstration | Keep `test_curl.sh` simple | No test framework or elaborate harness |
| Go sample | Leave `cmd/sample` unchanged | Curl is the only requested client demonstration |
| Authorization folder | Leave unchanged | Explicitly out of scope |
| Replicas | Single Caddy instance | Distributed rate limiting is out of scope |

## Configuration interface

Add `rules.txt` with one rule per line:

```text
# domain-pattern requests-per-second
*.sec.gov 10
*.yahoo.com 2
api.example.com 5
```

Rules:

- Blank lines and lines beginning with `#` are ignored.
- The rate must be a positive integer representing requests per rolling second.
- `*.sec.gov` includes the apex domain and subdomains at any depth.
- All hosts covered by a wildcard rule consume the same rate-limit bucket.
- `api.example.com` matches that exact host only.
- Duplicate and overlapping rules are rejected with a clear generator error.
- At least one rule is required.

## HTTP interface

### Health check

```http
GET /healthz
```

Expected response:

```http
HTTP/1.1 200 OK
```

The health check does not require authentication or consume a rate-limit event.

### Proxied request

```http
GET /proxy/data.sec.gov/submissions/example.json?format=json
Authorization: Basic <credentials>
User-Agent: Example Company admin@example.com
```

Caddy will send:

```http
GET https://data.sec.gov/submissions/example.json?format=json
```

The proxy will preserve:

- HTTP method
- Request body
- Original path
- Query string
- User-Agent
- Content headers and other ordinary request headers

The proxy's Basic `Authorization` header will be consumed and removed before forwarding. Supporting a separate upstream `Authorization` header is out of scope.

### Error behavior

| Condition | Status | Behavior |
|---|---:|---|
| Invalid or missing proxy credentials | `401` | Basic-authentication challenge |
| Malformed `/proxy/` URL | `400` | Clear invalid-target message |
| Target has no matching rule | `403` | `target domain is not configured: <host>` |
| Local limit exceeded | `429` | Clear rate-limit message and `Retry-After` header |
| Upstream failure | Upstream status or `502` | Do not retry automatically |
| Valid request | Upstream status | Pass upstream response through |

## Generator changes

Update `cmd/generate` to:

1. Accept `-in rules.txt` and `-out Caddyfile`.
2. Parse and validate the rule file.
3. Generate stable, deterministic route and zone names.
4. Generate one static rate-limit bucket per rule.
5. Generate matchers for exact domains or wildcard domain groups.
6. Configure HTTPS upstream routing using the validated host as:
   - The upstream address
   - TLS SNI
   - The upstream `Host` header
7. Add a final rejection route for unconfigured targets.
8. Add a generated-file warning at the top of the Caddyfile.
9. Exit nonzero without replacing a valid Caddyfile when input is invalid.

The generated routing order must be explicit:

```text
health check
→ authentication
→ allowed-domain match
→ rate limit
→ path rewrite
→ HTTPS reverse proxy
→ reject unmatched target
```

## Container and command changes

- Continue using Docker Compose and host port `8090`.
- Pin builder and runtime images to Caddy `2.11.9`.
- Build Caddy with `github.com/mholt/caddy-ratelimit`.
- Remove the unused `forwardproxy` plugin from the custom build.
- Keep the Caddyfile bind mount.
- Correct the password-hash environment substitution.
- Keep `.env` ignored by Git.
- Retain the current credential generation performed by `make up`.
- Add:

  ```text
  make generate
  ```

- Make startup behave as:

  ```text
  make up
  → generate Caddyfile
  → validate prerequisites
  → start Docker Compose
  ```

- Do not modify `authorization/` or `cmd/sample/`.
- Preserve the user's existing uncommitted `cmd/app` to `cmd/sample` changes.

## Documentation changes

Expand the README to document:

- Why a reverse proxy is used instead of a standard HTTPS forward proxy.
- The `rules.txt` format.
- Wildcard aggregation semantics.
- The `/proxy/<host>/<path>` URL contract.
- Authentication setup.
- `make generate`, `make up`, `make down`, and `make test`.
- Status codes and local `429` behavior.
- That the proxy rejects rather than queues excess traffic.
- How calling applications rewrite upstream URLs.
- SEC's User-Agent requirement.
- Cloud deployments must add trusted TLS termination and network restrictions.

SEC currently specifies a total maximum of 10 requests per second and asks automated callers to declare an identifying User-Agent: [SEC fair-access guidance](https://www.sec.gov/search-filings/edgar-search-assistance/accessing-edgar-data).

## Testing

### Generator tests

Cover:

- A valid exact-domain rule.
- A valid wildcard rule.
- Comments and blank lines.
- Zero, negative, missing, and nonnumeric rates.
- Invalid domains and wildcard placement.
- Duplicate rules.
- Overlapping exact and wildcard rules.
- Empty input.
- Stable output for identical input.

### Caddy validation

- Generate the Caddyfile from `rules.txt`.
- Validate it using the custom Caddy binary.
- Confirm Caddy starts successfully in Docker Compose.

### Curl demonstration

Keep `test_curl.sh` readable and demonstrate:

1. `/healthz` returns `200`.
2. Missing credentials are rejected.
3. An authenticated allowed request reaches its upstream.
4. Paths and query strings are preserved.
5. Requests to an unconfigured domain return the proxy's `403`.
6. A small concurrent burst exceeds a low test rule and returns local `429` responses.
7. `Retry-After` is present on local `429` responses.
8. Requests to different SEC subdomains consume one shared bucket.
9. SEC requests include a declared User-Agent.

Concurrency is necessary in the burst demonstration because sequential external requests may take longer than one second and fail to trigger the configured limit.

## Acceptance criteria

- Every request reaching an allowed upstream consumes exactly one event.
- A rule allowing `N` requests per second accepts no more than `N` requests in any rolling one-second window.
- Apex and subdomain requests covered by one wildcard rule share the same counter.
- Excess traffic is rejected locally with `429` and never reaches the upstream.
- Unconfigured domains never reach an upstream.
- Multiple configured domain groups use independent limits.
- Generated configuration passes Caddy validation.
- The container is reachable on host port `8090`.
- `test_curl.sh` clearly demonstrates allowed, throttled, unauthorized, and forbidden requests.
- The authorization folder and Go sample remain unchanged.

## Assumptions

- Only one Caddy container is running. Multiple replicas would each maintain an independent counter.
- Limits use Caddy's sliding-window implementation.
- Upstreams use public HTTPS with valid certificates.
- Upstream authentication is not required.
- The proxy does not provide retries, backoff, queuing, or request pacing.
- Basic authentication over plain HTTP is suitable only on localhost or a trusted private network. A cloud-facing deployment must terminate TLS before traffic reaches this listener.
