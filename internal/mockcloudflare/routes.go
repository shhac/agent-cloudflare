package mockcloudflare

func Routes() []string {
	return []string{
		"GET /healthz",
		"GET /user/tokens/verify",
		"GET /accounts",
		"GET /accounts/{account_id}/rulesets",
		"GET /accounts/{account_id}/waiting_rooms",
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
