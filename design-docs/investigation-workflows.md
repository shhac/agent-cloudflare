# Investigation workflows

Investigation commands answer incident-shaped questions with compact evidence records and findings. They do not mutate Cloudflare state.

| Command | Primary evidence | Use when |
| --- | --- | --- |
| `investigate zone-health <zone>` | Zone, DNS, SSL/TLS, cache, rulesets, Waiting Rooms | General zone health triage |
| `investigate traffic-spike <zone> --since 1h` | GraphQL traffic analytics, audit context | Request volume, 5xx, or traffic anomaly |
| `investigate dns-change <zone>` | DNS summary, audit context | DNS changed or resolving unexpectedly |
| `investigate ssl-breakage <zone>` | SSL/TLS zone settings | HTTPS/certificate symptoms |
| `investigate waf-block <zone>` | Ruleset summary | Users appear blocked or challenged |
| `investigate worker-error --account-id <acct>` | Worker inventory | Worker-related 5xx or deployment concern |
| `investigate cache-miss <zone>` | Cache API and zone settings | Poor cache hit ratio or unexpected origin load |

Evidence record shape:

```json
{"type":"entity","object":"zone","id":"...","data":{}}
{"type":"finding","severity":"warning","summary":"...","data":{}}
```

Cloudflare analytics availability depends on plan, token permission, dataset, and time window. Analytics failures should keep useful `fixable_by` hints.
