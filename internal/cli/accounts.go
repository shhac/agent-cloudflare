package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func registerAccounts(root *cobra.Command, globals shared.GlobalsFunc) {
	accounts := &cobra.Command{
		Use:   "accounts",
		Short: "Read Cloudflare accounts visible to the active token",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				params := url.Values{}
				shared.AddString(params, "id", resolved.AccountID)
				items, info, err := client.Accounts(ctx, params)
				if err != nil {
					return err
				}
				return shared.WriteRawPaginatedList(items, info, flags.Format)
			})
		},
	}
	accounts.AddCommand(list)
	root.AddCommand(accounts)
}
