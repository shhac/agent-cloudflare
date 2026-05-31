package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func registerZoneSettings(root *cobra.Command, globals shared.GlobalsFunc) {
	settings := &cobra.Command{
		Use:   "zone-settings",
		Short: "Read Cloudflare zone settings",
	}
	get := &cobra.Command{
		Use:   "get <setting-id> [zone-name-or-id]",
		Short: "Get one zone setting",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				zoneRef := resolved.ZoneID
				if len(args) > 1 {
					zoneRef = args[1]
				}
				zoneID, err := resolveZoneID(ctx, client, resolved, zoneRef)
				if err != nil {
					return err
				}
				raw, err := client.ZoneSetting(ctx, zoneID, args[0])
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, flags.Format)
				return nil
			})
		},
	}
	settings.AddCommand(get)
	root.AddCommand(settings)
}
