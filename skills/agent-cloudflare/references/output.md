# agent-cloudflare output and errors

Use this reference when interpreting command output, explaining failures, or deciding whether a command is safe to run.

## Output Modes

Lists default to NDJSON so agents can stream and process one row at a time. Single resources default to JSON. Commands that produce investigation evidence use NDJSON records with a `type` field.

Example evidence rows:

```json
{"type":"entity","object":"zone","id":"...","data":{}}
{"type":"finding","severity":"warning","summary":"...","data":{}}
```

Finding severities are `info`, `warning`, and `critical`.

## Error Contract

Errors are machine-readable JSON on stderr. They include a stable code, a human-facing message, `fixable_by`, and usually a `hint`.

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
