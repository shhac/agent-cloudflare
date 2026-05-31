---
name: agent-cloudflare
description: |
  Investigate Cloudflare accounts, zones, DNS, SSL/TLS, rulesets, cache, Workers, KV, R2, Waiting Rooms, and traffic/security issues using a secret-safe CLI. Use when:
  - Checking Cloudflare zone configuration or DNS records
  - Investigating Cloudflare traffic, cache, WAF, rules, SSL/TLS, or Waiting Room state
  - Looking up account/zone IDs for Cloudflare resources
  - Making authenticated Cloudflare API reads without exposing API tokens
  Triggers: "cloudflare", "cf zone", "dns record", "waf", "ruleset", "cache purge", "waiting room", "worker", "workers kv", "r2 bucket", "cloudflare api"
allowed-tools: Bash(agent-cloudflare *) Bash(mockcloudflare *) Read Grep Glob
---

# agent-cloudflare

Use `agent-cloudflare` for Cloudflare operations triage and read-only resource inspection.

## Safety

- Never ask the tool to reveal an API token.
- Never accept pasted Cloudflare tokens in chat. Ask the user to run `agent-cloudflare profiles add <profile> --form` locally so the token goes directly into an OS dialog.
- Use `agent-cloudflare profiles update <profile> --form` when a stored token needs replacement.
- Prefer read-only commands.
- Use `--account-id` and `--zone-id` to scope commands when multiple accounts or zones are visible.
- Treat mutations such as DNS changes or cache purges as high stakes. Use `--dry-run` first and only use `--confirm` when the user explicitly asks for the write.

## Start Here

```bash
agent-cloudflare usage
agent-cloudflare profiles list
agent-cloudflare profiles check
agent-cloudflare profiles discover <profile> --zone example.com
agent-cloudflare config show
agent-cloudflare accounts list
agent-cloudflare zones list
agent-cloudflare investigate usage
```

For direct exploration:

```bash
agent-cloudflare zones get example.com
agent-cloudflare dns list example.com --type A
agent-cloudflare ssl status example.com
agent-cloudflare cache settings example.com
agent-cloudflare rulesets list example.com
agent-cloudflare waiting-rooms list example.com
agent-cloudflare workers list --account-id <account_id>
agent-cloudflare workers get <script_name> --account-id <account_id>
agent-cloudflare kv namespaces list --account-id <account_id>
agent-cloudflare r2 buckets list --account-id <account_id>
agent-cloudflare audit list --account-id <account_id>
agent-cloudflare analytics traffic example.com --since 1h
agent-cloudflare investigate zone-health example.com
agent-cloudflare investigate traffic-spike example.com --since 1h
agent-cloudflare investigate dns-change example.com
agent-cloudflare investigate ssl-breakage example.com
agent-cloudflare investigate waf-block example.com
agent-cloudflare investigate worker-error --account-id <account_id>
agent-cloudflare investigate cache-miss example.com
agent-cloudflare snapshot zone example.com
agent-cloudflare api get /zones --query name=example.com
agent-cloudflare api get /zones/<zone_id>/dns_records --query type=CNAME
```

For mutation previews:

```bash
agent-cloudflare cache purge example.com --url https://example.com/a --dry-run
agent-cloudflare dns create example.com --type CNAME --name app --content target.example.com --dry-run
agent-cloudflare waiting-rooms update wr_... example.com --enabled --dry-run
```

For local testing, run `mockcloudflare` and set `--base-url http://127.0.0.1:12112` with `--api-token cfut_mock`.

## Output

Lists default to NDJSON. Single resources default to JSON. Errors include `fixable_by` and usually a `hint`.

Investigation output uses evidence records:

```json
{"type":"entity","object":"zone","id":"...","data":{}}
{"type":"finding","severity":"warning","summary":"...","data":{}}
```

Profile/config metadata lives in XDG config. API tokens live in Keychain. `profiles list` and `profiles check` may show non-secret credential type (`cfut`, `legacy_api_token`, or `unknown`) but never the token.

## Incremental References

Load these only when useful:

- [references/scenarios.md](references/scenarios.md): common incident questions and recommended command sequences.
