# agent-cloudflare output and errors

Use this reference when interpreting command output, explaining failures, or deciding whether a command is safe to run.

## Output Modes

Lists default to NDJSON so agents can stream and process one row at a time.

**Get (single + multi).** `get <id>...` takes one or more ids and returns one result per id, in input order. Default output is NDJSON: one line per id — the record, or `{"@unresolved":{"id","reason","fixable_by","hint"?}}` for an id that couldn't be resolved (e.g. not found / bad id). `--format json|yaml` collapses to one `{"data":[…],"@unresolved":[…]}` envelope. A single `get <id>` is just the one-element case (NDJSON one line by default; was pretty JSON before — pass `--format json` for the object). Item-level misses stay on stdout and exit 0; only a command-level failure (auth, network) goes to stderr with exit 1 and empty stdout.

**Cloudflare-specific scope flag.** `zone-settings get <setting-id>...` and `waiting-rooms get <waiting-room-id>...` take zone scope via `--zone <zone-name-or-id>` (a flag, not a trailing positional argument). `api get` stays single (raw escape hatch, not an entity get).

Commands that produce investigation evidence use NDJSON records with a `type` field.

Example evidence rows:

```json
{"type":"entity","object":"zone","id":"...","data":{}}
{"type":"finding","severity":"warning","summary":"...","data":{}}
```

Finding severities are `info`, `warning`, and `critical`.

## Error Contract

Errors are a single JSON object on stderr: `{"error":"...","fixable_by":"agent"|"human"|"retry","hint"?:"...","retry_after_seconds"?:N}` — exit 1. `hint` and `retry_after_seconds` are optional.

Common `fixable_by` meanings:

- `agent`: change the command, add a missing flag, narrow the account/zone, or correct an ID/name.
- `human`: the user needs to complete setup, grant permissions, replace credentials, or approve a mutation.
- `retry`: network, rate limit, temporary Cloudflare failure, or response-shape issue where retrying or narrowing the request may help.

When reporting an error to the user, include the useful hint and the next command to try. Do not expose secrets or ask the user to paste a token.

## Secrets

Profile metadata lives in XDG config. API tokens live in Keychain. `profiles list` and `profiles check` may show non-secret credential type (`cfut`, `legacy_api_token`, or `unknown`) but never token values.

Use `agent-cloudflare profiles add <profile> --form` or `agent-cloudflare profiles update <profile> --form` for token entry so the LLM never sees the secret.

## Mutation Approval

Mutation commands are guarded by explicit flags:

- `--dry-run`: show the planned operation without calling the write endpoint.
- `--confirm`: perform the write after explicit user approval.

For mutation workflows, run dry-run first, summarize the planned change and blast radius, then wait for explicit user approval before confirm.
