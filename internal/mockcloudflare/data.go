package mockcloudflare

func accounts() []map[string]any {
	return []map[string]any{
		{
			"id":   "023e105f4ecef8ad9ca31a8372d0c353",
			"name": "Mock Production",
			"type": "standard",
		},
		{
			"id":   "11111111111111111111111111111111",
			"name": "Mock Sandbox",
			"type": "standard",
		},
	}
}

func zones() []map[string]any {
	return []map[string]any{
		{
			"id":     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"name":   "example.com",
			"status": "active",
			"account": map[string]any{
				"id":   "023e105f4ecef8ad9ca31a8372d0c353",
				"name": "Mock Production",
			},
			"type": "full",
		},
		{
			"id":     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"name":   "staging.example.com",
			"status": "pending",
			"account": map[string]any{
				"id":   "11111111111111111111111111111111",
				"name": "Mock Sandbox",
			},
			"type": "partial",
		},
	}
}

func dnsRecords(zoneID string) []map[string]any {
	if zoneID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"id":        "dns_mock_a",
			"type":      "A",
			"name":      "example.com",
			"content":   "203.0.113.10",
			"proxied":   true,
			"ttl":       1,
			"proxiable": true,
		},
		{
			"id":        "dns_mock_www",
			"type":      "CNAME",
			"name":      "www.example.com",
			"content":   "example.com",
			"proxied":   true,
			"ttl":       1,
			"proxiable": true,
		},
	}
}

func zoneSetting(zoneID, settingID string) (map[string]any, bool) {
	if zoneID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		return nil, false
	}
	settings := map[string]map[string]any{
		"ssl": {
			"id":       "ssl",
			"value":    "full",
			"editable": true,
		},
		"always_use_https": {
			"id":       "always_use_https",
			"value":    "off",
			"editable": true,
		},
		"automatic_https_rewrites": {
			"id":       "automatic_https_rewrites",
			"value":    "on",
			"editable": true,
		},
		"min_tls_version": {
			"id":       "min_tls_version",
			"value":    "1.2",
			"editable": true,
		},
		"tls_1_3": {
			"id":       "tls_1_3",
			"value":    "on",
			"editable": true,
		},
		"ssl_recommender": {
			"id":      "ssl_recommender",
			"enabled": true,
		},
		"cache_level": {
			"id":       "cache_level",
			"value":    "aggressive",
			"editable": true,
		},
		"browser_cache_ttl": {
			"id":       "browser_cache_ttl",
			"value":    14400,
			"editable": true,
		},
		"development_mode": {
			"id":             "development_mode",
			"value":          "off",
			"editable":       true,
			"time_remaining": 0,
		},
		"always_online": {
			"id":       "always_online",
			"value":    "on",
			"editable": true,
		},
		"brotli": {
			"id":       "brotli",
			"value":    "on",
			"editable": true,
		},
	}
	setting, ok := settings[settingID]
	return setting, ok
}

func cacheSetting(zoneID, path string) (map[string]any, bool) {
	if zoneID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		return nil, false
	}
	settings := map[string]map[string]any{
		"cache_reserve": {
			"id":       "cache_reserve",
			"value":    "off",
			"editable": true,
		},
		"tiered_cache_smart_topology_enable": {
			"id":       "tiered_cache_smart_topology_enable",
			"value":    "on",
			"editable": true,
		},
		"regional_tiered_cache": {
			"id":       "tc_regional",
			"value":    "off",
			"editable": true,
		},
	}
	setting, ok := settings[path]
	return setting, ok
}

func zoneRulesets(zoneID string) []map[string]any {
	if zoneID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"id":    "ruleset_mock_waf",
			"name":  "zone managed WAF",
			"kind":  "zone",
			"phase": "http_request_firewall_managed",
			"rules": []map[string]any{
				{"id": "rule_mock_execute", "enabled": true, "action": "execute"},
			},
		},
		{
			"id":    "ruleset_mock_transform",
			"name":  "response headers",
			"kind":  "zone",
			"phase": "http_response_headers_transform",
			"rules": []map[string]any{
				{"id": "rule_mock_header", "enabled": false, "action": "rewrite"},
			},
		},
	}
}

func accountRulesets(accountID string) []map[string]any {
	if accountID != "023e105f4ecef8ad9ca31a8372d0c353" {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"id":    "ruleset_mock_account",
			"name":  "account redirect rules",
			"kind":  "root",
			"phase": "http_request_redirect",
		},
	}
}

func zoneWaitingRooms(zoneID string) []map[string]any {
	if zoneID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"id":          "wr_mock_sale",
			"name":        "sale-room",
			"description": "Mock launch room",
			"host":        "example.com",
			"path":        "/sale",
			"enabled":     true,
			"queue_all":   false,
		},
	}
}

func accountWaitingRooms(accountID string) []map[string]any {
	if accountID != "023e105f4ecef8ad9ca31a8372d0c353" {
		return []map[string]any{}
	}
	return zoneWaitingRooms("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
}

func waitingRoom(zoneID, roomID string) (map[string]any, bool) {
	for _, room := range zoneWaitingRooms(zoneID) {
		if room["id"] == roomID || room["name"] == roomID {
			return room, true
		}
	}
	return nil, false
}

func filterByString(items []map[string]any, field, want string) []map[string]any {
	out := []map[string]any{}
	for _, item := range items {
		if got, _ := item[field].(string); got == want {
			out = append(out, item)
		}
	}
	return out
}

func filterByNestedString(items []map[string]any, parent, field, want string) []map[string]any {
	out := []map[string]any{}
	for _, item := range items {
		nested, _ := item[parent].(map[string]any)
		if got, _ := nested[field].(string); got == want {
			out = append(out, item)
		}
	}
	return out
}
