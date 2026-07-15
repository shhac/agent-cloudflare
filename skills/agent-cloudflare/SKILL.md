---
name: agent-cloudflare
description: |
  Read-first Cloudflare operations triage for AI agents. Covers accounts, zones, DNS, SSL/TLS, rulesets, cache, Workers, KV, R2, Waiting Rooms, audit logs, analytics, snapshots, and guarded mutations, all via a secret-safe CLI. Use when:
  - Checking Cloudflare zone configuration or DNS records
  - Investigating Cloudflare traffic, cache, WAF/rulesets, SSL/TLS, Waiting Room, Worker, KV, R2, or audit-log context
  - Looking up account/zone IDs for Cloudflare resources
  - Making authenticated Cloudflare reads or dry-run mutation previews without exposing API tokens
  Triggers: "cloudflare", "cf zone", "dns record", "waf", "ruleset", "cache purge", "waiting room", "worker", "workers kv", "r2 bucket", "cloudflare api"
allowed-tools: Bash(agent-cloudflare *) Bash(mockcloudflare *) Read Grep Glob
---

# agent-cloudflare

Use `agent-cloudflare` for Cloudflare operations triage. Prefer investigation commands for incident-shaped questions and direct resource commands when the user names a specific Cloudflare resource.

## Safety

- Never ask the tool to reveal an API token.
- Never accept pasted Cloudflare tokens in chat. Ask the user to run `agent-cloudflare profiles add <profile> --form` locally so the token goes directly into an OS dialog.
- Use `agent-cloudflare profiles update <profile> --form` when a stored token needs replacement.
- Prefer read-only commands.
- Use `--account-id` and `--zone-id` to scope commands when multiple accounts or zones are visible.
- Treat mutations such as DNS changes or cache purges as high stakes. Use `--dry-run` first and only use `--confirm` when the user explicitly asks for the write.

## Choose The Path

- Setup or credential issue: start with `usage`, `profiles list`, `profiles check`, and `profiles discover`.
- Incident, outage, or vague question: start with `investigate usage`, then choose the closest investigation.
- Named resource or known ID: use the matching resource command from `references/commands.md`.
- Write request: use the dry-run command first, report the planned operation, then wait for explicit approval before `--confirm`.
- Unsupported Cloudflare endpoint: use `api get` for authenticated reads and keep output scoped.

## First Commands

```bash
agent-cloudflare usage
agent-cloudflare profiles list
agent-cloudflare profiles check
agent-cloudflare investigate usage
```

For local testing, run `mockcloudflare` and set `--base-url http://127.0.0.1:12112` with `--api-token cfut_mock`.

## Output Contract

Lists default to NDJSON. **Get (single + multi).** `get <id>...` accepts one or more ids and returns one result per id, in input order. Default output is NDJSON: one line per id — the record, or `{"@unresolved":{"id","reason","fixable_by","hint"?}}` for an id that couldn't be resolved (e.g. not found). `--format json|yaml` collapses to one `{"data":[…],"@unresolved":[…]}` envelope. A single `get <id>` is the one-element case (NDJSON by default; pass `--format json` for the object). Item-level misses stay on stdout and exit 0; only command-level failures (auth, network) go to stderr with exit 1. `zone-settings get` and `waiting-rooms get` scope their zone via `--zone <zone-name-or-id>` (not a trailing positional). `api get` stays single. Errors include `fixable_by` and usually a `hint`.

Investigation output uses evidence records:

```json
{"type":"entity","object":"zone","id":"...","data":{}}
{"type":"finding","severity":"warning","summary":"...","data":{}}
```

Profile/config metadata lives in XDG config. API tokens live in Keychain. `profiles list` and `profiles check` may show non-secret credential type (`cfut`, `legacy_api_token`, or `unknown`) but never the token.

## Incremental References

Load these only when useful:

- [references/scenarios.md](references/scenarios.md): common incident questions and recommended command sequences.
- [references/commands.md](references/commands.md): command chooser and exact syntax for resources, investigations, raw API reads, and guarded mutations.
- [references/output.md](references/output.md): NDJSON/JSON conventions, error hints, evidence rows, secret handling, and mutation approval rules.
