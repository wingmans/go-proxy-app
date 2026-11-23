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

   When `.env` is missing, Make copies the working local values from
   `.env.sample`. The sample starts the included mock API automatically and
   authenticates upstream as `test-user:test-password` using HTTP Basic.
   Callers authenticate separately as
   `test-proxy-user:test-proxy-password`; only the caller password hash enters
   the Caddy container.

2. To target another API, edit `.env`. `API_UPSTREAM` must be an origin such as
   `https://api.github.com`, without a path.

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

## Curl validation

`test.sh` assumes the authorization proxy and its upstream are already running.
It does not start, stop, or remove containers. The script reads its settings
from `.env`, calls
the health endpoint, and then sends an explicit curl request containing caller
credentials that must be replaced by the configured upstream credential.

Start the complete sample stack:

```sh
make up
```

Run the curl script:

```sh
./test.sh
```

Stop the stack explicitly when finished:

```sh
make down
```

## Security boundary

Anyone who can reach this proxy can use its upstream credential. Docker
Compose therefore publishes it on `127.0.0.1` only. Before exposing it on a
network, add authentication or mTLS for callers and restrict which upstream
paths and methods they may use. Keep `.env` out of version control.
