# Cloudflare docs alignment

Docs checked on 2026-05-31:

- Cloudflare API Reference: `https://developers.cloudflare.com/api/`
- Create API token: `https://developers.cloudflare.com/fundamentals/api/get-started/create-token/`
- Zones API: `https://developers.cloudflare.com/api/resources/zones/`
- Accounts API: `https://developers.cloudflare.com/api/resources/accounts/`
- DNS API: `https://developers.cloudflare.com/api/resources/dns/`
- Zone Settings API: `https://developers.cloudflare.com/api/node/resources/zones/subresources/settings/`
- Cache API: `https://developers.cloudflare.com/api/resources/cache/`
- Rulesets API: `https://developers.cloudflare.com/api/resources/rulesets/methods/list/`
- Waiting Rooms API: `https://developers.cloudflare.com/api/resources/waiting_rooms/`
- Workers API: `https://developers.cloudflare.com/api/resources/workers/`
- KV API: `https://developers.cloudflare.com/api/resources/kv/`
- R2 API: `https://developers.cloudflare.com/api/resources/r2/`

## API shape

Cloudflare's REST API is rooted at:

```text
https://api.cloudflare.com/client/v4
```

Most endpoints return an envelope with `success`, `result`, `errors`, `messages`, and often `result_info`. `agent-cloudflare` unwraps `result` for ordinary command output and converts `result_info` into an NDJSON `@pagination` row.

## Authentication

Cloudflare recommends API tokens over legacy Global API Keys. The token creation docs describe both user tokens and Account API tokens:

- user tokens are created from My Profile > API Tokens
- Account API tokens are created from Manage Account > API Tokens
- token permissions can be scoped by permission group and resource
- `/user/tokens/verify` validates the current token
- newer token secrets use the `cfut_` prefix

`agent-cloudflare` should prefer token auth:

```http
Authorization: Bearer <API_TOKEN>
```

Legacy email plus Global API Key auth may be added later only if a real workflow needs it. It should not be the default because it encourages broader credentials.

## Scope model

Cloudflare resources are a mix of:

- user-visible account lists: `/accounts`
- account-scoped resources: `/accounts/{account_id}/...`
- zone-scoped resources: `/zones/{zone_id}/...`
- analytics through REST and GraphQL surfaces

This is why profile metadata stores both account and zone defaults. A token can be valid but still fail a command if its permission group or resource scope excludes the requested account or zone.

## Initial endpoint mapping

```text
profiles check       GET /user/tokens/verify
accounts list        GET /accounts
zones list           GET /zones
zones get <id>       GET /zones/{zone_id}
zones get <name>     GET /zones?name=<name>, then GET /zones/{zone_id}
dns list <zone>      GET /zones/{zone_id}/dns_records
zone-settings get    GET /zones/{zone_id}/settings/{setting_id}
ssl status           GET selected /zones/{zone_id}/settings/{setting_id}
cache settings       GET /zones/{zone_id}/cache/{setting} plus selected zone settings
rulesets list        GET /zones/{zone_id}/rulesets or /accounts/{account_id}/rulesets
waiting-rooms list   GET /zones/{zone_id}/waiting_rooms or /accounts/{account_id}/waiting_rooms
waiting-rooms get    GET /zones/{zone_id}/waiting_rooms/{waiting_room_id}
workers list         GET /accounts/{account_id}/workers/scripts
workers get          GET /accounts/{account_id}/workers/scripts/{script_name}/subdomain
                     and /accounts/{account_id}/workers/scripts/{script_name}/versions
kv namespaces list   GET /accounts/{account_id}/storage/kv/namespaces
kv namespaces get    GET /accounts/{account_id}/storage/kv/namespaces/{namespace_id}
r2 buckets list      GET /accounts/{account_id}/r2/buckets
r2 buckets get       GET /accounts/{account_id}/r2/buckets/{bucket_name}
investigate zone-health
                     combines zone, DNS, SSL/TLS, cache, rulesets, and Waiting Rooms reads
api get <path>       GET <path>
```

## Future endpoint families

The docs show broad families worth wrapping in read-first order:

- DNS records and DNSSEC under `/zones/{zone_id}/dns...`
- Zone settings such as SSL/TLS, HTTP/3, Brotli, development mode, WAF, cache, and security level
- Rulesets, WAF, bot, Access, Turnstile, Waiting Rooms, and load balancers
- Workers under `/accounts/{account_id}/workers...`
- KV namespaces under `/accounts/{account_id}/storage/kv/namespaces`
- R2 buckets under `/accounts/{account_id}/r2/buckets`

Analytics needs separate design because plan availability and GraphQL permissions vary.
