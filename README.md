# agent-cloudflare

Cloudflare operations CLI for AI agents.

`agent-cloudflare` is read-first and secret-safe by default. It stores Cloudflare API tokens outside model-visible output, emits structured JSON errors, and defaults list output to NDJSON so agents can stream and filter results.

## Quick Start

```bash
agent-cloudflare profiles add prod --form --account-id <account_id>
agent-cloudflare profiles check prod
agent-cloudflare accounts list
agent-cloudflare zones list
agent-cloudflare dns list example.com
agent-cloudflare investigate zone-health example.com
```

`profiles` is the canonical credential command group. `auth` is supported as a hidden compatibility alias.

For local fixture testing:

```bash
make build
make build-mock
./mockcloudflare
./agent-cloudflare --base-url http://127.0.0.1:12112 --api-token cfut_mock accounts list
./agent-cloudflare --base-url http://127.0.0.1:12112 --api-token cfut_mock investigate zone-health example.com
```

## Development

```bash
make test
go vet ./...
make build
make build-mock
```

Design notes live in [design-docs/initial-design.md](design-docs/initial-design.md).
