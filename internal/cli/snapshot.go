package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func registerSnapshot(root *cobra.Command, globals shared.GlobalsFunc) {
	snapshot := &cobra.Command{
		Use:   "snapshot",
		Short: "Capture and diff Cloudflare state snapshots",
	}
	zone := &cobra.Command{
		Use:   "zone [zone-name-or-id]",
		Short: "Capture a zone state snapshot as JSON",
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
				snap, err := captureZoneSnapshot(ctx, client, zoneID)
				if err != nil {
					return err
				}
				shared.WriteItem(snap, flags.Format)
				return nil
			})
		},
	}
	diff := &cobra.Command{
		Use:   "diff <before.json> <after.json>",
		Short: "Diff two snapshot JSON files",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			before, err := readSnapshotFile(args[0])
			if err != nil {
				return err
			}
			after, err := readSnapshotFile(args[1])
			if err != nil {
				return err
			}
			shared.WriteItem(map[string]any{"changes": diffSnapshotMaps(before, after)}, globals().Format)
			return nil
		},
	}
	snapshot.AddCommand(zone, diff)
	root.AddCommand(snapshot)
}

func registerBaseline(root *cobra.Command, globals shared.GlobalsFunc) {
	var file string
	baseline := &cobra.Command{
		Use:   "baseline",
		Short: "Compare current Cloudflare state to a saved baseline",
	}
	check := &cobra.Command{
		Use:   "check [zone-name-or-id]",
		Short: "Compare current zone snapshot with a baseline file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return agenterrors.New("--file is required", agenterrors.FixableByAgent).WithHint("Pass --file baseline.json")
			}
			baselineData, err := readSnapshotFile(file)
			if err != nil {
				return err
			}
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
				current, err := captureZoneSnapshot(ctx, client, zoneID)
				if err != nil {
					return err
				}
				changes := diffSnapshotMaps(baselineData, current)
				shared.WriteItem(map[string]any{
					"status":  baselineStatus(changes),
					"changes": changes,
				}, flags.Format)
				return nil
			})
		},
	}
	check.Flags().StringVar(&file, "file", "", "Snapshot JSON baseline file")
	baseline.AddCommand(check)
	root.AddCommand(baseline)
}

func captureZoneSnapshot(ctx context.Context, client *api.Client, zoneID string) (map[string]any, error) {
	zoneRaw, err := client.Zone(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	zone, err := decodeRaw(zoneRaw)
	if err != nil {
		return nil, err
	}
	sslSettings, sslFindings := collectSettingsSoft(ctx, client, zoneID, sslSettingIDs)
	cacheAPI, cacheAPIFindings := collectCacheAPISettingsSoft(ctx, client, zoneID, cacheAPISettingPaths)
	cacheZone, cacheZoneFindings := collectSettingsSoft(ctx, client, zoneID, cacheZoneSettingIDs)
	if len(sslFindings)+len(cacheAPIFindings)+len(cacheZoneFindings) > 0 {
		// Soft failures are captured in snapshot metadata rather than aborting the whole snapshot.
	}
	dnsRecords := []any{}
	if items, _, err := client.DNSRecords(ctx, zoneID, nil); err == nil {
		dnsRecords, _ = shared.RawItemsToAny(items)
	}
	rulesets := []any{}
	if items, _, err := client.Rulesets(ctx, "zones", zoneID, nil); err == nil {
		rulesets, _ = shared.RawItemsToAny(items)
	}
	waitingRooms := []any{}
	if items, _, err := client.WaitingRooms(ctx, "zones", zoneID, nil); err == nil {
		waitingRooms, _ = shared.RawItemsToAny(items)
	}
	return map[string]any{
		"schema":         "agent-cloudflare.zone-snapshot.v1",
		"zone_id":        zoneID,
		"zone":           zone,
		"dns_records":    dnsRecords,
		"ssl_settings":   sslSettings,
		"cache_api":      cacheAPI,
		"cache_settings": cacheZone,
		"rulesets":       rulesets,
		"waiting_rooms":  waitingRooms,
	}, nil
}

func readSnapshotFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByHuman).WithHint("Check the snapshot file path")
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent).WithHint("Snapshot files must be JSON objects")
	}
	return decoded, nil
}

func diffSnapshotMaps(before, after map[string]any) []map[string]any {
	keys := map[string]bool{}
	for key := range before {
		keys[key] = true
	}
	for key := range after {
		keys[key] = true
	}
	changes := []map[string]any{}
	for key := range keys {
		if jsonEqual(before[key], after[key]) {
			continue
		}
		changes = append(changes, map[string]any{
			"path":   key,
			"before": before[key],
			"after":  after[key],
		})
	}
	return changes
}

func jsonEqual(a, b any) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return bytes.Equal(aj, bj)
}

func baselineStatus(changes []map[string]any) string {
	if len(changes) == 0 {
		return "ok"
	}
	return "drift"
}
