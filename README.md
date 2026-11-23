# Caddy proxy experiments

This repository contains two independent Caddy experiments:

- [`authorization`](./authorization): a reverse proxy that injects a private
  API credential into upstream requests.
- [`throttling`](./throttling): a loopback HTTP gateway that allowlists and
  rate-limits requests before forwarding them to HTTPS services.

Each directory has its own Caddy configuration, Docker Compose stack, tests,
and usage instructions. Run commands from the directory for the experiment
you want to use.

List the commands for each proxy from the repository root:

```sh
make -C throttling help
make -C authorization help
```

For example:

```sh
make -C throttling up
make -C throttling down
```

The root Makefile only points to these proxy-specific Makefiles. It does not
start or coordinate containers.
