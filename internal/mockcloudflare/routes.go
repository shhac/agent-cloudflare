package mockcloudflare

func Routes() []string {
	return []string{
		"GET /healthz",
		"POST /graphql",
		"GET /user/tokens/verify",
		"GET /accounts",
		"GET /accounts/{account_id}/logs/audit",
		"GET /accounts/{account_id}/rulesets",
		"GET /accounts/{account_id}/waiting_rooms",
		"GET /accounts/{account_id}/workers/scripts",
		"GET /accounts/{account_id}/workers/scripts/{script_name}/subdomain",
		"GET /accounts/{account_id}/workers/scripts/{script_name}/versions",
		"GET /accounts/{account_id}/storage/kv/namespaces",
		"GET /accounts/{account_id}/storage/kv/namespaces/{namespace_id}",
		"GET /accounts/{account_id}/r2/buckets",
		"GET /accounts/{account_id}/r2/buckets/{bucket_name}",
		"GET /zones",
		"GET /zones/{zone_id}",
		"GET /zones/{zone_id}/dns_records",
		"GET /zones/{zone_id}/settings/{setting_id}",
		"GET /zones/{zone_id}/cache/{setting}",
		"GET /zones/{zone_id}/rulesets",
		"GET /zones/{zone_id}/waiting_rooms",
		"GET /zones/{zone_id}/waiting_rooms/{waiting_room_id}",
	}
}
