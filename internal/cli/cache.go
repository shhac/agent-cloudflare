package cli

import (
	"context"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
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
	registerCachePurge(cache, globals)
	root.AddCommand(cache)
}

func registerCachePurge(parent *cobra.Command, globals shared.GlobalsFunc) {
	var urls []string
	var everything bool
	var dryRun bool
	var confirm bool
	var reason string

	cmd := &cobra.Command{
		Use:   "purge [zone-name-or-id]",
		Short: "Purge Cloudflare cache with dry-run or explicit confirmation",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireMutationMode(dryRun, confirm); err != nil {
				return err
			}
			if everything == (len(urls) > 0) {
				return agenterrors.New("choose exactly one purge target", agenterrors.FixableByAgent).
					WithHint("Pass --url one or more times, or pass --everything")
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
				body := map[string]any{}
				if everything {
					body["purge_everything"] = true
				} else {
					body["files"] = urls
				}
				if reason != "" {
					body["reason"] = reason
				}
				path := "/zones/" + zoneID + "/purge_cache"
				if dryRun {
					writeDryRun(client, flags, http.MethodPost, path, body)
					return nil
				}
				raw, err := client.PurgeCache(ctx, zoneID, body)
				if err != nil {
					return err
				}
				decoded, err := decodeRaw(raw)
				if err != nil {
					return err
				}
				shared.WriteItem(mutationResult("cache.purge", decoded), flags.Format)
				return nil
			})
		},
	}
	cmd.Flags().StringArrayVar(&urls, "url", nil, "URL to purge; repeatable")
	cmd.Flags().BoolVar(&everything, "everything", false, "Purge the entire zone cache")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the request without sending it")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Send the mutation")
	cmd.Flags().StringVar(&reason, "reason", "", "Human-readable reason for audit context")
	parent.AddCommand(cmd)
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
