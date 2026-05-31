package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func registerRulesets(root *cobra.Command, globals shared.GlobalsFunc) {
	var account bool

	rulesets := &cobra.Command{
		Use:   "rulesets",
		Short: "Read Cloudflare rulesets",
	}
	list := &cobra.Command{
		Use:   "list [zone-name-or-id]",
		Short: "List zone or account rulesets",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				scope, scopeID, err := resolveAccountOrZoneScope(ctx, client, resolved, account, args)
				if err != nil {
					return err
				}
				items, info, err := client.Rulesets(ctx, scope, scopeID, nil)
				if err != nil {
					return err
				}
				decoded, err := shared.RawItemsToAny(items)
				if err != nil {
					return err
				}
				shared.WritePaginatedList(decoded, info, flags.Format)
				return nil
			})
		},
	}
	list.Flags().BoolVar(&account, "account", false, "List account-level rulesets using --account-id or profile default account")
	rulesets.AddCommand(list)
	root.AddCommand(rulesets)
}

func resolveAccountOrZoneScope(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, account bool, args []string) (string, string, error) {
	if account {
		if resolved.AccountID == "" {
			return "", "", agenterrors.New("--account-id is required for account-scoped rulesets", agenterrors.FixableByAgent).
				WithHint("Pass --account-id or store one with profiles update")
		}
		return "accounts", resolved.AccountID, nil
	}
	zoneRef := resolved.ZoneID
	if len(args) > 0 {
		zoneRef = args[0]
	}
	zoneID, err := resolveZoneID(ctx, client, resolved, zoneRef)
	if err != nil {
		return "", "", err
	}
	return "zones", zoneID, nil
}
