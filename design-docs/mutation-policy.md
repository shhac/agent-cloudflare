# Mutation policy

`agent-cloudflare` is read-only in the current implementation.

Future write commands must meet these requirements before they are documented in the skill:

- write commands use explicit verbs such as `purge`, `create`, `update`, or `delete`
- every write supports a dry-run or request-preview mode
- live writes require an explicit `--confirm` flag
- investigation commands never mutate Cloudflare state
- output includes structured request intent and result details
- `mockcloudflare` covers the write path before the command is released
- errors stay JSON with `fixable_by` and actionable hints

Candidate future write commands:

```text
agent-cloudflare cache purge <zone> --url https://example.com/path --dry-run
agent-cloudflare cache purge <zone> --url https://example.com/path --confirm
agent-cloudflare dns create <zone> --type CNAME --name app --content target.example.com --dry-run
agent-cloudflare waiting-rooms update <zone> <room> --enabled true --dry-run
```

No write command should accept a pasted token, print a token, or infer confirmation from an LLM-authored prompt.
