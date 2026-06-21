# agent-cloudflare initial design

## Goal

Build a Go CLI that makes Cloudflare operations triage easy for an LLM while keeping Cloudflare API tokens out of model-visible output.

The first version is read-first:

- identify the active credential profile and non-secret account/zone defaults
- list accounts visible to the token
- list and retrieve zones
- list DNS records
- inspect SSL/TLS, cache settings, rulesets, and Waiting Rooms
- gather zone-health investigation evidence
- provide a GET-only raw API escape hatch for endpoints not yet wrapped
- ship with `mockcloudflare` for fixture-driven e2e tests

The CLI is not a replacement for `wrangler`. `wrangler` is developer-platform/deploy focused; this CLI is for operational inspection, evidence gathering, and agent-safe support workflows.

## Credential and profile model

Use `profile` as the user-facing concept. Cloudflare credentials can be tied to different things:

- legacy Global API Keys are user-level and require email-based auth, so this CLI does not optimize for them
- user API tokens are tied to a Cloudflare user and scoped by permissions/resources
- Account API tokens are account-level service tokens for compatible endpoints

Because both user tokens and account tokens can operate over account and zone resources, the durable object the CLI manages is a profile: a Keychain-stored API token plus non-secret defaults.

`profiles` is the canonical command group:

```text
agent-cloudflare profiles add <profile> --form [--account-id <acct>] [--zone-id <zone>] [--zone example.com]
agent-cloudflare profiles update <profile> [--form|--api-token <token>] [--account-id <acct>] [--zone-id <zone>] [--zone example.com]
agent-cloudflare profiles check [profile]
agent-cloudflare profiles list
agent-cloudflare profiles default <profile>
agent-cloudflare profiles remove <profile>
```

`auth` is a hidden alias for sibling-project familiarity, but docs should teach `profiles`.

Non-secret metadata lives in `${XDG_CONFIG_HOME}/agent-cloudflare/config.json`, or `~/.config/agent-cloudflare/config.json` when `XDG_CONFIG_HOME` is unset:

- profile alias
- default account ID and optional account label
- default zone ID and optional zone name
- zone-name to zone-ID cache
- non-secret credential classification, such as `cfut`, `legacy_api_token`, or `unknown`

The token itself is stored in macOS Keychain. A `credentials.json` index records that a profile is Keychain-managed, but never stores token values. The credential package has no list/print secret method.

LLM-led setup should use:

```bash
agent-cloudflare profiles add prod --form
agent-cloudflare profiles update prod --form
```

The native dialog asks for the token outside the terminal/chat context. The CLI prints only a redacted receipt.

## Output contract

Lists default to NDJSON so an LLM can stream, truncate, and resume without parsing large arrays.

**Get (single + multi).** Entity `get` commands accept `get <id>...` (one or more ids) and emit one result per id, in input order, as NDJSON: the record, or `{"@unresolved":{"id","reason","fixable_by","hint"?}}` for an id that couldn't be resolved (not found, bad id). `--format json|yaml` collapses all results into a `{"data":[…],"@unresolved":[…]}` envelope. A single `get <id>` is the one-element case — NDJSON by default (was pretty JSON before; pass `--format json` for the object). Item-level misses stay on stdout and exit 0; only command-level failures (auth, network, bad flag) write a `{error}` JSON object to stderr and exit 1.

**Cloudflare-specific: zone scope as flag.** `zone-settings get <setting-id>...` and `waiting-rooms get <waiting-room-id>...` scope their zone via `--zone <zone-name-or-id>` (a flag). The trailing positional `[zone]` pattern was removed to support multi-id gets. `api get` stays single (raw escape hatch, not an entity get).

Errors are JSON on stderr:

```json
{"error":"Permission denied: ...","fixable_by":"human","hint":"The token may need ..."}
```

`fixable_by` is one of:

- `agent`: malformed command, wrong ID shape, missing flag, bad query construction
- `human`: credential, permission, account membership, or dashboard setup issue
- `retry`: network, rate limit, or Cloudflare server issue

Debug output uses structured JSON records on stderr and must not include raw tokens.

## Command surface

Initial wrapped commands:

```text
agent-cloudflare usage
agent-cloudflare profiles add|update|check|list|default|remove
agent-cloudflare profiles discover
agent-cloudflare config show|path|set|unset
agent-cloudflare accounts list
agent-cloudflare zones list|get
agent-cloudflare dns list
agent-cloudflare zone-settings get
agent-cloudflare ssl status
agent-cloudflare cache settings
agent-cloudflare rulesets list
agent-cloudflare waiting-rooms list|get
agent-cloudflare workers list|get
agent-cloudflare kv namespaces list|get
agent-cloudflare r2 buckets list|get
agent-cloudflare audit list
agent-cloudflare analytics traffic
agent-cloudflare investigate usage|zone-health|traffic-spike|dns-change|ssl-breakage|waf-block|worker-error|cache-miss
agent-cloudflare snapshot zone|diff
agent-cloudflare baseline check
agent-cloudflare api get
```

Near-term read-only additions:

```text
agent-cloudflare analytics graphql-query --file query.graphql
agent-cloudflare investigate worker-error <script-name>
```

## Investigation model

Investigation commands emit NDJSON evidence records:

```json
{"type":"entity","object":"traffic_analytics","id":"...","data":{}}
{"type":"finding","severity":"warning","summary":"...","data":{}}
```

Current investigations:

- `zone-health`: zone, DNS, SSL/TLS, cache, rulesets, and Waiting Room posture
- `traffic-spike`: GraphQL traffic analytics plus audit context
- `dns-change`: DNS summary plus audit context
- `ssl-breakage`: SSL/TLS setting evidence and warnings
- `waf-block`: rulesets summary
- `worker-error`: account Worker inventory evidence
- `cache-miss`: cache API and zone settings

## Snapshot model

`snapshot zone` captures repeatable JSON state for a zone. `snapshot diff` compares two files at top-level snapshot paths. `baseline check` compares current state with a saved snapshot and returns `ok` or `drift`.

## Mutation policy

The CLI is read-first. Mutations live behind explicit verbs and confirmation or dry-run gates:

```text
agent-cloudflare cache purge <zone> --url ...
agent-cloudflare dns create <zone> --type CNAME --name app --content ...
agent-cloudflare dns update <zone> <record-id> --content ...
agent-cloudflare waiting-rooms update <zone> <room> --enabled true
```

Mutation command requirements:

- dry-run or preview output before sending a write request
- explicit `--confirm` flag for live writes
- no implicit mutation from investigation commands
- structured before/after or request-preview output
- e2e coverage in `mockcloudflare` before exposing the command in the skill

## Testing pattern

Code should stay easy to test through dependency injection:

- API clients accept base URL, token, HTTP client hooks when needed
- credential storage is behind a small backend interface
- dialog prompting is behind a `Prompter`
- command tests can point `--base-url` at `mockcloudflare`
- e2e tests should use fixture routes instead of live Cloudflare

## Mock server

`mockcloudflare` implements a small Cloudflare v4-style envelope:

```json
{"success":true,"result":{},"errors":[],"messages":[],"result_info":{}}
```

It covers accounts, zones, token verification, DNS records, zone settings, cache settings, rulesets, and Waiting Rooms. As command coverage grows, tests should extend mock routes before or alongside new CLI wrappers.
