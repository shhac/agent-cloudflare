# Snapshots and baselines

Snapshots give agents a reviewable representation of Cloudflare state.

Capture a zone:

```bash
agent-cloudflare snapshot zone example.com > zone-snapshot.json
```

Diff two snapshots:

```bash
agent-cloudflare snapshot diff before.json after.json
```

Check current state against a baseline:

```bash
agent-cloudflare baseline check example.com --file zone-snapshot.json
```

Current snapshot schema:

```json
{
  "schema": "agent-cloudflare.zone-snapshot.v1",
  "zone_id": "...",
  "zone": {},
  "dns_records": [],
  "ssl_settings": {},
  "cache_api": {},
  "cache_settings": {},
  "rulesets": [],
  "waiting_rooms": []
}
```

Future improvements:

- stable sorting for arrays before diffing
- path-level nested diffs
- snapshot redaction controls if new resources include sensitive fields
- named baseline directories per profile/account/zone
