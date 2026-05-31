package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

var cacheAPISettingPaths = []string{
	"cache_reserve",
	"tiered_cache_smart_topology_enable",
	"regional_tiered_cache",
}

var cacheZoneSettingIDs = []string{
	"cache_level",
	"browser_cache_ttl",
	"development_mode",
	"always_online",
	"brotli",
}

func registerCache(root *cobra.Command, globals shared.GlobalsFunc) {
	cache := &cobra.Command{
		Use:   "cache",
		Short: "Read Cloudflare cache posture for a zone",
	}
	settings := &cobra.Command{
		Use:   "settings [zone-name-or-id]",
		Short: "Summarize cache-related settings",
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
				apiSettings, err := collectCacheAPISettings(ctx, client, zoneID, cacheAPISettingPaths)
				if err != nil {
					return err
				}
				zoneSettings, err := collectSettings(ctx, client, zoneID, cacheZoneSettingIDs)
				if err != nil {
					return err
				}
				shared.WriteItem(map[string]any{
					"zone_id":       zoneID,
					"cache_api":     apiSettings,
					"zone_settings": zoneSettings,
				}, flags.Format)
				return nil
			})
		},
	}
	cache.AddCommand(settings)
	root.AddCommand(cache)
}

func collectCacheAPISettings(ctx context.Context, client *api.Client, zoneID string, settingPaths []string) (map[string]any, error) {
	settings := map[string]any{}
	for _, path := range settingPaths {
		raw, err := client.CacheSetting(ctx, zoneID, path)
		if err != nil {
			return nil, err
		}
		decoded, err := decodeRaw(raw)
		if err != nil {
			return nil, err
		}
		settings[path] = decoded
	}
	return settings, nil
}
