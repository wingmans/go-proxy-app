# Caddy proxy experiments

This repository contains two independent Caddy experiments:

- [`authorization`](./authorization): a reverse proxy that injects a private
  API credential into upstream requests.
- [`throttling`](./throttling): an authenticated forward proxy that applies
  per-host request limits.

Each directory has its own Caddy configuration, Docker Compose stack, tests,
and usage instructions. Run commands from the directory for the experiment
you want to use.
