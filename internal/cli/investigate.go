package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	"github.com/shhac/agent-cloudflare/internal/output"
)

type evidenceRecord struct {
	Type     string `json:"type"`
	Object   string `json:"object,omitempty"`
	ID       string `json:"id,omitempty"`
	Severity string `json:"severity,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Data     any    `json:"data,omitempty"`
}

func registerInvestigate(root *cobra.Command, globals shared.GlobalsFunc) {
	var since string

	investigate := &cobra.Command{
		Use:   "investigate",
		Short: "Gather Cloudflare evidence for common operational questions",
	}
	investigate.AddCommand(&cobra.Command{
		Use:   "usage",
		Short: "Show investigation command examples",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), `agent-cloudflare investigate usage

Available investigations:
  agent-cloudflare investigate zone-health <zone-name-or-id>
  agent-cloudflare investigate traffic-spike <zone-name-or-id> --since 1h
  agent-cloudflare investigate dns-change <zone-name-or-id>
  agent-cloudflare investigate ssl-breakage <zone-name-or-id>
  agent-cloudflare investigate waf-block <zone-name-or-id>
  agent-cloudflare investigate worker-error --account-id <account_id>
  agent-cloudflare investigate cache-miss <zone-name-or-id>

Output:
  Investigation records default to NDJSON evidence rows.
  Finding rows use severity: info, warning, critical.
`)
			return nil
		},
	})
	zoneHealth := &cobra.Command{
		Use:   "zone-health [zone-name-or-id]",
		Short: "Gather zone, DNS, SSL/TLS, cache, rulesets, and Waiting Room evidence",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				zoneRef := resolved.ZoneID
				if len(args) > 0 {
					zoneRef = args[0]
				}
				zoneID, err := resolveZoneID(ctx, client, resolved, zoneRef)
				if err != nil {
					return err
				}
				records := investigateZoneHealth(ctx, client, zoneID)
				writeEvidence(records, flags.Format)
				return nil
			})
		},
	}
	investigate.AddCommand(zoneHealth)

	trafficSpike := &cobra.Command{
		Use:   "traffic-spike [zone-name-or-id]",
		Short: "Gather analytics and audit evidence for a traffic spike",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZoneInvestigation(globals(), args, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
				start, end, err := sinceWindow(since, time.Now())
				if err != nil {
					return nil, err
				}
				return investigateTrafficSpike(ctx, client, resolved, zoneID, start, end)
			})
		},
	}
	trafficSpike.Flags().StringVar(&since, "since", "1h", "Lookback duration, such as 15m, 1h, or 24h")

	investigate.AddCommand(trafficSpike)
	investigate.AddCommand(zoneInvestigationCommand("dns-change", "Gather DNS and audit evidence for recent DNS changes", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
		return investigateDNSChange(ctx, client, resolved, zoneID), nil
	}))
	investigate.AddCommand(zoneInvestigationCommand("ssl-breakage", "Gather SSL/TLS evidence for certificate or HTTPS issues", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
		settings, findings := collectSettingsSoft(ctx, client, zoneID, sslSettingIDs)
		records := []evidenceRecord{{Type: "entity", Object: "ssl_settings", ID: zoneID, Data: settings}}
		records = append(records, findings...)
		records = append(records, sslFindings(settings)...)
		return records, nil
	}))
	investigate.AddCommand(zoneInvestigationCommand("waf-block", "Gather rulesets and traffic evidence for suspected WAF blocks", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
		return investigateWAFBlock(ctx, client, zoneID), nil
	}))
	investigate.AddCommand(zoneInvestigationCommand("cache-miss", "Gather cache settings and traffic evidence for cache misses", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
		return investigateCacheMiss(ctx, client, zoneID), nil
	}))
	investigate.AddCommand(accountInvestigationCommand("worker-error", "Gather Workers evidence for account-level Worker errors", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, accountID string) ([]evidenceRecord, error) {
		return investigateWorkerError(ctx, client, accountID), nil
	}))
	root.AddCommand(investigate)
}

type zoneInvestigationFunc func(context.Context, *api.Client, *shared.ResolvedProfile, string) ([]evidenceRecord, error)
type accountInvestigationFunc func(context.Context, *api.Client, *shared.ResolvedProfile, string) ([]evidenceRecord, error)

func zoneInvestigationCommand(use, short string, globals shared.GlobalsFunc, fn zoneInvestigationFunc) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [zone-name-or-id]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZoneInvestigation(globals(), args, fn)
		},
	}
}

func accountInvestigationCommand(use, short string, globals shared.GlobalsFunc, fn accountInvestigationFunc) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				accountID, err := requireAccountID(resolved)
				if err != nil {
					return err
				}
				records, err := fn(ctx, client, resolved, accountID)
				if err != nil {
					return err
				}
				writeEvidence(records, flags.Format)
				return nil
			})
		},
	}
}

func runZoneInvestigation(flags *shared.GlobalFlags, args []string, fn zoneInvestigationFunc) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
		zoneRef := resolved.ZoneID
		if len(args) > 0 {
			zoneRef = args[0]
		}
		zoneID, err := resolveZoneID(ctx, client, resolved, zoneRef)
		if err != nil {
			return err
		}
		records, err := fn(ctx, client, resolved, zoneID)
		if err != nil {
			return err
		}
		writeEvidence(records, flags.Format)
		return nil
	})
}

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

func writeEvidence(records []evidenceRecord, format string) {
	if output.ResolveFormat(format, output.FormatNDJSON) == output.FormatNDJSON {
		w := output.NewNDJSONWriter(output.Stdout())
		for _, record := range records {
			_ = w.WriteItem(record)
		}
		return
	}
	shared.WriteItem(map[string]any{"records": records}, format)
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

func errorFinding(object, severity, summary string, err error) evidenceRecord {
	return evidenceRecord{
		Type:     "finding",
		Object:   object,
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"error": strings.TrimSpace(err.Error()),
		},
	}
}

func asMap(value any) map[string]any {
	m, _ := value.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func stringValue(m map[string]any, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}

func boolValue(m map[string]any, key string) bool {
	value, _ := m[key].(bool)
	return value
}

func intNumber(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func prettyJSON(value any) string {
	b, _ := json.Marshal(value)
	return string(b)
}
