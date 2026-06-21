package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func registerZoneSettings(root *cobra.Command, globals shared.GlobalsFunc) {
	var zone string

	settings := &cobra.Command{
		Use:   "zone-settings",
		Short: "Read Cloudflare zone settings",
	}
	get := &cobra.Command{
		Use:   "get <setting-id>... [--zone <zone-name-or-id>]",
		Short: "Get one or more zone settings",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.GetEntities(flags, args, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, id string) (any, error) {
				zoneRef := zone
				if zoneRef == "" {
					zoneRef = resolved.ZoneID
				}
				zoneID, err := resolveZoneID(ctx, client, resolved, zoneRef)
				if err != nil {
					return nil, err
				}
				raw, err := client.ZoneSetting(ctx, zoneID, id)
				if err != nil {
					return nil, err
				}
				return decodeRaw(raw)
			})
		},
	}
	get.Flags().StringVar(&zone, "zone", "", "Zone name or ID (overrides profile default)")
	settings.AddCommand(get)
	root.AddCommand(settings)
}
