# agent-cloudflare Initial Idea

## Premise

Cloudflare has a broad API, but the official `wrangler` CLI is primarily focused on developer-platform workflows such as Workers, Pages, D1, R2, KV, Queues, Durable Objects, and secrets. There is room for a read-first, agent-safe Cloudflare operations CLI focused on zone investigation, traffic context, DNS, SSL, rules, cache, and Waiting Rooms.

## Gap

`wrangler` is excellent for building and deploying Cloudflare developer resources, but it is not a general-purpose Cloudflare ops triage CLI. An AI agent often needs to answer questions like:

- Is this zone configured correctly?
- What changed in DNS or rules recently?
- Is traffic spiking or being blocked?
- Is SSL/TLS healthy?
- Are Waiting Rooms configured and active?
- Can we inspect cache, rulesets, WAF, bot, or Access state without exposing secrets?

The Cloudflare API exposes much of this, but raw API calls are too open-ended and not shaped for safe LLM use.

## Product Shape

`agent-cloudflare` should be a Cloudflare operations and investigation CLI for AI agents.

Design principles:

- Read-first by default.
- Structured output to stdout, structured errors to stderr.
- Token-efficient list output, likely NDJSON by default.
- Secret-safe credential profiles stored outside LLM context.
- Explicit opt-in for mutations such as cache purge, DNS changes, or Waiting Room updates.
- Classified errors with `fixable_by: agent|human|retry`.
- Narrow investigation commands that gather evidence instead of dumping entire Cloudflare objects.

## Candidate Scope

Initial command groups:

```bash
agent-cloudflare profiles add prod --form
agent-cloudflare profiles check prod
agent-cloudflare zones list
agent-cloudflare zones get example.com
agent-cloudflare dns list example.com
agent-cloudflare ssl status example.com
agent-cloudflare analytics traffic example.com --since 1h
agent-cloudflare waiting-rooms list example.com
agent-cloudflare waiting-rooms get example.com sale-room
agent-cloudflare rulesets list example.com
agent-cloudflare cache status example.com
agent-cloudflare investigate zone-health example.com
agent-cloudflare investigate traffic-spike example.com --since 1h
```

Potential mutation commands, behind explicit flags or confirmation gates:

```bash
agent-cloudflare cache purge example.com --url https://example.com/path
agent-cloudflare dns create example.com --type CNAME --name app --content target.example.com
agent-cloudflare waiting-rooms update example.com sale-room --enabled true
```

## Differentiator

The tool is not a replacement for `wrangler`. It fills the gap between raw Cloudflare API access and Worker-focused deployment tooling:

- Operations triage over developer deployment.
- Safer defaults for LLM delegation.
- Compact evidence records instead of dashboard-shaped payloads.
- High-level investigations across zones, DNS, SSL, rules, analytics, and Waiting Rooms.

## Open Questions

- Should v1 be strictly read-only, with mutations added later? Current answer:
  yes, except explicit future mutation groups with separate confirmation gates.
- Which Cloudflare analytics APIs are available for typical account plans?
- How should account, zone, and profile defaults be modeled? Current answer:
  a profile stores one secret token reference plus non-secret default account and zone IDs.
- Should Waiting Rooms be first-class in v1, or follow after core zone/DNS/SSL/analytics support?
- Should the CLI support both REST and GraphQL Analytics APIs behind one command shape?
