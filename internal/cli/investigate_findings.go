package cli

func trafficFindings(data any) []evidenceRecord {
	findings := []evidenceRecord{}
	total, errors := trafficCounts(data)
	if total > 0 && errors*100/total >= 5 {
		findings = append(findings, evidenceRecord{
			Type:     "finding",
			Severity: "warning",
			Summary:  "Elevated 5xx response share in traffic analytics",
			Data:     map[string]any{"requests": total, "server_errors": errors},
		})
	}
	return findings
}

func trafficCounts(data any) (total, serverErrors int) {
	viewer := asMap(asMap(data)["data"])
	viewer = asMap(viewer["viewer"])
	zones, _ := viewer["zones"].([]any)
	for _, zone := range zones {
		series, _ := asMap(zone)["series"].([]any)
		for _, row := range series {
			m := asMap(row)
			count := intNumber(m["count"])
			total += count
			status := intNumber(asMap(m["dimensions"])["edgeResponseStatus"])
			if status >= 500 && status <= 599 {
				serverErrors += count
			}
		}
	}
	return total, serverErrors
}

func sslFindings(settings map[string]any) []evidenceRecord {
	findings := []evidenceRecord{}
	if ssl := settingValue(settings, "ssl"); ssl != "" && ssl != "strict" {
		findings = append(findings, evidenceRecord{
			Type:     "finding",
			Severity: "warning",
			Summary:  "SSL mode is not Full (strict)",
			Data:     map[string]any{"ssl": ssl},
		})
	}
	if alwaysHTTPS := settingValue(settings, "always_use_https"); alwaysHTTPS == "off" {
		findings = append(findings, evidenceRecord{
			Type:     "finding",
			Severity: "warning",
			Summary:  "Always Use HTTPS is off",
			Data:     map[string]any{"always_use_https": alwaysHTTPS},
		})
	}
	if rewrites := settingValue(settings, "automatic_https_rewrites"); rewrites == "off" {
		findings = append(findings, evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary:  "Automatic HTTPS Rewrites is off",
			Data:     map[string]any{"automatic_https_rewrites": rewrites},
		})
	}
	return findings
}

func cacheFindings(settings map[string]any) []evidenceRecord {
	findings := []evidenceRecord{}
	if devMode := settingValue(settings, "development_mode"); devMode == "on" {
		findings = append(findings, evidenceRecord{
			Type:     "finding",
			Severity: "warning",
			Summary:  "Development Mode is on",
			Data:     map[string]any{"development_mode": devMode},
		})
	}
	return findings
}

func dnsSummary(records []any) map[string]any {
	byType := map[string]int{}
	proxied := 0
	unproxied := 0
	for _, record := range records {
		m := asMap(record)
		recordType := stringValue(m, "type")
		if recordType == "" {
			recordType = "unknown"
		}
		byType[recordType]++
		if boolValue(m, "proxied") {
			proxied++
		} else {
			unproxied++
		}
	}
	return map[string]any{
		"total":     len(records),
		"by_type":   byType,
		"proxied":   proxied,
		"unproxied": unproxied,
	}
}

func dnsFindings(summary map[string]any) []evidenceRecord {
	findings := []evidenceRecord{}
	total, _ := summary["total"].(int)
	if total == 0 {
		findings = append(findings, evidenceRecord{
			Type:     "finding",
			Severity: "warning",
			Summary:  "No DNS records were returned for this zone",
		})
	}
	return findings
}

func rulesetsSummary(items []any) map[string]any {
	byPhase := map[string]int{}
	enabledRules := 0
	totalRules := 0
	for _, item := range items {
		m := asMap(item)
		phase := stringValue(m, "phase")
		if phase == "" {
			phase = "unknown"
		}
		byPhase[phase]++
		if rules, ok := m["rules"].([]any); ok {
			totalRules += len(rules)
			for _, rule := range rules {
				if boolValue(asMap(rule), "enabled") {
					enabledRules++
				}
			}
		}
	}
	return map[string]any{
		"total":         len(items),
		"by_phase":      byPhase,
		"total_rules":   totalRules,
		"enabled_rules": enabledRules,
	}
}

func waitingRoomsSummary(items []any) map[string]any {
	enabled := 0
	for _, item := range items {
		if boolValue(asMap(item), "enabled") {
			enabled++
		}
	}
	return map[string]any{
		"total":    len(items),
		"enabled":  enabled,
		"disabled": len(items) - enabled,
	}
}

func settingValue(settings map[string]any, key string) string {
	m := asMap(settings[key])
	return stringValue(m, "value")
}
