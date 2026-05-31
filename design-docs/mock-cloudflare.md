# mockcloudflare

`mockcloudflare` is a local Cloudflare API fixture server for e2e tests and agent demos.

Run it:

```bash
make mock
```

Then point the CLI at it:

```bash
agent-cloudflare --base-url http://127.0.0.1:12112 --api-token cfut_mock accounts list
agent-cloudflare --base-url http://127.0.0.1:12112 --api-token cfut_mock zones list
agent-cloudflare --base-url http://127.0.0.1:12112 --api-token cfut_mock dns list example.com
agent-cloudflare --base-url http://127.0.0.1:12112 --api-token cfut_mock --account-id 023e105f4ecef8ad9ca31a8372d0c353 workers list
agent-cloudflare --base-url http://127.0.0.1:12112 --api-token cfut_mock --account-id 023e105f4ecef8ad9ca31a8372d0c353 kv namespaces list
agent-cloudflare --base-url http://127.0.0.1:12112 --api-token cfut_mock --account-id 023e105f4ecef8ad9ca31a8372d0c353 r2 buckets list
agent-cloudflare --base-url http://127.0.0.1:12112 --api-token cfut_mock investigate zone-health example.com
```

Implemented routes:

```text
GET /healthz
GET /user/tokens/verify
GET /accounts
GET /accounts/{account_id}/rulesets
GET /accounts/{account_id}/waiting_rooms
GET /accounts/{account_id}/workers/scripts
GET /accounts/{account_id}/workers/scripts/{script_name}/subdomain
GET /accounts/{account_id}/workers/scripts/{script_name}/versions
GET /accounts/{account_id}/storage/kv/namespaces
GET /accounts/{account_id}/storage/kv/namespaces/{namespace_id}
GET /accounts/{account_id}/r2/buckets
GET /accounts/{account_id}/r2/buckets/{bucket_name}
GET /zones
GET /zones/{zone_id}
GET /zones/{zone_id}/dns_records
GET /zones/{zone_id}/settings/{setting_id}
GET /zones/{zone_id}/cache/{setting}
GET /zones/{zone_id}/rulesets
GET /zones/{zone_id}/waiting_rooms
GET /zones/{zone_id}/waiting_rooms/{waiting_room_id}
```

The server requires a non-empty `Authorization: Bearer ...` header for API routes and returns Cloudflare-style v4 envelopes.
