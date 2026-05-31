package cli

import (
	"context"
	"time"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func investigateZoneHealth(ctx context.Context, client *api.Client, zoneID string) []evidenceRecord {
	records := []evidenceRecord{}

	zoneRaw, err := client.Zone(ctx, zoneID)
	if err != nil {
		return append(records, errorFinding("zone", "critical", "Could not retrieve zone", err))
	}
	zone, err := decodeRaw(zoneRaw)
	if err != nil {
		return append(records, errorFinding("zone", "critical", "Could not decode zone", err))
	}
	records = append(records, evidenceRecord{Type: "entity", Object: "zone", ID: zoneID, Data: zone})
	zoneMap := asMap(zone)
	if status := stringValue(zoneMap, "status"); status != "" && status != "active" {
		records = append(records, evidenceRecord{
			Type:     "finding",
			Severity: "warning",
			Summary:  "Zone is not active",
			Data:     map[string]any{"status": status},
		})
	}

	sslSettings, sslErrs := collectSettingsSoft(ctx, client, zoneID, sslSettingIDs)
	records = append(records, evidenceRecord{Type: "entity", Object: "ssl_settings", ID: zoneID, Data: sslSettings})
	records = append(records, sslErrs...)
	records = append(records, sslFindings(sslSettings)...)

	cacheAPI, cacheAPIErrs := collectCacheAPISettingsSoft(ctx, client, zoneID, cacheAPISettingPaths)
	cacheZone, cacheZoneErrs := collectSettingsSoft(ctx, client, zoneID, cacheZoneSettingIDs)
	records = append(records, evidenceRecord{Type: "entity", Object: "cache_settings", ID: zoneID, Data: map[string]any{
		"cache_api":     cacheAPI,
		"zone_settings": cacheZone,
	}})
	records = append(records, cacheAPIErrs...)
	records = append(records, cacheZoneErrs...)
	records = append(records, cacheFindings(cacheZone)...)

	dnsItems, _, err := client.DNSRecords(ctx, zoneID, nil)
	if err != nil {
		records = append(records, errorFinding("dns_records", "warning", "Could not retrieve DNS records", err))
	} else {
		dnsRecords, err := shared.RawItemsToAny(dnsItems)
		if err != nil {
			records = append(records, errorFinding("dns_records", "warning", "Could not decode DNS records", err))
		} else {
			summary := dnsSummary(dnsRecords)
			records = append(records, evidenceRecord{Type: "entity", Object: "dns_records_summary", ID: zoneID, Data: summary})
			records = append(records, dnsFindings(summary)...)
		}
	}

	rulesets, _, err := client.Rulesets(ctx, "zones", zoneID, nil)
	if err != nil {
		records = append(records, errorFinding("rulesets", "info", "Could not retrieve zone rulesets", err))
	} else {
		decoded, err := shared.RawItemsToAny(rulesets)
		if err != nil {
			records = append(records, errorFinding("rulesets", "info", "Could not decode zone rulesets", err))
		} else {
			records = append(records, evidenceRecord{Type: "entity", Object: "rulesets_summary", ID: zoneID, Data: rulesetsSummary(decoded)})
		}
	}

	waitingRooms, _, err := client.WaitingRooms(ctx, "zones", zoneID, nil)
	if err != nil {
		records = append(records, errorFinding("waiting_rooms", "info", "Could not retrieve Waiting Rooms", err))
	} else {
		decoded, err := shared.RawItemsToAny(waitingRooms)
		if err != nil {
			records = append(records, errorFinding("waiting_rooms", "info", "Could not decode Waiting Rooms", err))
		} else {
			summary := waitingRoomsSummary(decoded)
			records = append(records, evidenceRecord{Type: "entity", Object: "waiting_rooms_summary", ID: zoneID, Data: summary})
		}
	}

	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary:  "Zone health investigation complete",
		Data:     map[string]any{"zone_id": zoneID},
	})
	return records
}

func investigateTrafficSpike(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string, start, end time.Time) ([]evidenceRecord, error) {
	raw, err := client.GraphQL(ctx, trafficQuery, map[string]any{
		"zoneTag": zoneID,
		"start":   start.Format(time.RFC3339),
		"end":     end.Format(time.RFC3339),
		"limit":   200,
	})
	if err != nil {
		return nil, err
	}
	decoded, err := decodeRaw(raw)
	if err != nil {
		return nil, err
	}
	records := []evidenceRecord{
		{Type: "entity", Object: "traffic_analytics", ID: zoneID, Data: decoded},
	}
	records = append(records, trafficFindings(decoded)...)
	if resolved.AccountID != "" {
		auditRecords := auditEvidence(ctx, client, resolved.AccountID)
		records = append(records, auditRecords...)
	}
	return records, nil
}

func investigateDNSChange(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) []evidenceRecord {
	records := []evidenceRecord{}
	dnsItems, _, err := client.DNSRecords(ctx, zoneID, nil)
	if err != nil {
		records = append(records, errorFinding("dns_records", "warning", "Could not retrieve DNS records", err))
	} else if decoded, err := shared.RawItemsToAny(dnsItems); err == nil {
		records = append(records, evidenceRecord{Type: "entity", Object: "dns_records_summary", ID: zoneID, Data: dnsSummary(decoded)})
	}
	if resolved.AccountID != "" {
		records = append(records, auditEvidence(ctx, client, resolved.AccountID)...)
	}
	return records
}

func investigateWAFBlock(ctx context.Context, client *api.Client, zoneID string) []evidenceRecord {
	records := []evidenceRecord{}
	rulesets, _, err := client.Rulesets(ctx, "zones", zoneID, nil)
	if err != nil {
		return append(records, errorFinding("rulesets", "warning", "Could not retrieve WAF/rulesets", err))
	}
	decoded, err := shared.RawItemsToAny(rulesets)
	if err != nil {
		return append(records, errorFinding("rulesets", "warning", "Could not decode WAF/rulesets", err))
	}
	records = append(records, evidenceRecord{Type: "entity", Object: "rulesets_summary", ID: zoneID, Data: rulesetsSummary(decoded)})
	return records
}

func investigateCacheMiss(ctx context.Context, client *api.Client, zoneID string) []evidenceRecord {
	apiSettings, apiFindings := collectCacheAPISettingsSoft(ctx, client, zoneID, cacheAPISettingPaths)
	zoneSettings, zoneFindings := collectSettingsSoft(ctx, client, zoneID, cacheZoneSettingIDs)
	records := []evidenceRecord{{Type: "entity", Object: "cache_settings", ID: zoneID, Data: map[string]any{
		"cache_api":     apiSettings,
		"zone_settings": zoneSettings,
	}}}
	records = append(records, apiFindings...)
	records = append(records, zoneFindings...)
	records = append(records, cacheFindings(zoneSettings)...)
	return records
}

func investigateWorkerError(ctx context.Context, client *api.Client, accountID string) []evidenceRecord {
	items, _, err := client.Workers(ctx, accountID, nil)
	if err != nil {
		return []evidenceRecord{errorFinding("workers", "warning", "Could not retrieve Workers", err)}
	}
	decoded, err := shared.RawItemsToAny(items)
	if err != nil {
		return []evidenceRecord{errorFinding("workers", "warning", "Could not decode Workers", err)}
	}
	return []evidenceRecord{{Type: "entity", Object: "workers_summary", ID: accountID, Data: map[string]any{
		"total":   len(decoded),
		"workers": decoded,
	}}}
}

func auditEvidence(ctx context.Context, client *api.Client, accountID string) []evidenceRecord {
	items, info, err := client.AuditLogs(ctx, accountID, nil)
	if err != nil {
		return []evidenceRecord{errorFinding("audit_logs", "info", "Could not retrieve audit logs", err)}
	}
	decoded, err := shared.RawItemsToAny(items)
	if err != nil {
		return []evidenceRecord{errorFinding("audit_logs", "info", "Could not decode audit logs", err)}
	}
	return []evidenceRecord{{Type: "entity", Object: "audit_logs", ID: accountID, Data: map[string]any{
		"entries":    decoded,
		"pagination": info,
	}}}
}

func collectSettingsSoft(ctx context.Context, client *api.Client, zoneID string, settingIDs []string) (map[string]any, []evidenceRecord) {
	settings := map[string]any{}
	findings := []evidenceRecord{}
	for _, settingID := range settingIDs {
		raw, err := client.ZoneSetting(ctx, zoneID, settingID)
		if err != nil {
			findings = append(findings, errorFinding(settingID, "info", "Could not retrieve zone setting "+settingID, err))
			continue
		}
		decoded, err := decodeRaw(raw)
		if err != nil {
			findings = append(findings, errorFinding(settingID, "info", "Could not decode zone setting "+settingID, err))
			continue
		}
		settings[settingID] = decoded
	}
	return settings, findings
}

func collectCacheAPISettingsSoft(ctx context.Context, client *api.Client, zoneID string, settingPaths []string) (map[string]any, []evidenceRecord) {
	settings := map[string]any{}
	findings := []evidenceRecord{}
	for _, path := range settingPaths {
		raw, err := client.CacheSetting(ctx, zoneID, path)
		if err != nil {
			findings = append(findings, errorFinding(path, "info", "Could not retrieve cache setting "+path, err))
			continue
		}
		decoded, err := decodeRaw(raw)
		if err != nil {
			findings = append(findings, errorFinding(path, "info", "Could not decode cache setting "+path, err))
			continue
		}
		settings[path] = decoded
	}
	return settings, findings
}
