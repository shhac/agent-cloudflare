package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

var sslSettingIDs = []string{
	"ssl",
	"always_use_https",
	"automatic_https_rewrites",
	"min_tls_version",
	"tls_1_3",
	"ssl_recommender",
}

func registerSSL(root *cobra.Command, globals shared.GlobalsFunc) {
	ssl := &cobra.Command{
		Use:   "ssl",
		Short: "Read SSL/TLS posture for a zone",
	}
	status := &cobra.Command{
		Use:   "status [zone-name-or-id]",
		Short: "Summarize SSL/TLS zone settings",
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
				settings, err := collectSettings(ctx, client, zoneID, sslSettingIDs)
				if err != nil {
					return err
				}
				shared.WriteItem(map[string]any{
					"zone_id":  zoneID,
					"settings": settings,
				}, flags.Format)
				return nil
			})
		},
	}
	ssl.AddCommand(status)
	root.AddCommand(ssl)
}

func collectSettings(ctx context.Context, client *api.Client, zoneID string, settingIDs []string) (map[string]any, error) {
	settings := map[string]any{}
	for _, settingID := range settingIDs {
		raw, err := client.ZoneSetting(ctx, zoneID, settingID)
		if err != nil {
			return nil, err
		}
		decoded, err := decodeRaw(raw)
		if err != nil {
			return nil, err
		}
		settings[settingID] = decoded
	}
	return settings, nil
}
