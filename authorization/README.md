# Authorization proxy

This experiment uses Caddy as a local reverse proxy for a credential-protected
public API. A client calls Caddy without knowing the API credential; Caddy
replaces any caller-supplied credentials with the configured upstream
credential.

## Run it

1. Create the container and local environment file:

   ```sh
   make create
   ```

2. Set the public API origin and credential in `.env`. `API_UPSTREAM` must be
   an origin such as `https://api.github.com`, without a path.

3. Start the proxy. Running `make up` directly also creates `.env` when it is
   missing:

   ```sh
   make up
   ```

4. Call the upstream path through the proxy. With the GitHub example, this
   requests `https://api.github.com/user` while the client supplies no token:

   ```sh
   curl http://127.0.0.1:8091/user
   ```

For an API key, configure a custom header instead:

```dotenv
API_AUTH_HEADER=X-API-Key
API_AUTH_VALUE=replace-with-your-api-key
```

The proxy strips caller-provided `Authorization`, `Proxy-Authorization`, and
`Cookie` headers before adding the configured credential. Access logs do not
record request headers.

## Self-contained test

The integration test uses the normal Caddy proxy and `Caddyfile`. It adds a
private Go validation service as the upstream, then calls the published proxy
from the host. The validator returns success only when Caddy injected the
expected credential and stripped caller credentials.

Run both the curl and Go clients:

```sh
make test
```

Or run either host client independently:

```sh
make run-curl
make run-go
```

The test stack uses port `18091`, so it does not conflict with the normal
proxy on port `8091`. It is removed automatically when the test finishes.

## Security boundary

Anyone who can reach this proxy can use its upstream credential. Docker
Compose therefore publishes it on `127.0.0.1` only. Before exposing it on a
network, add authentication or mTLS for callers and restrict which upstream
paths and methods they may use. Keep `.env` out of version control.
