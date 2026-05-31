package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func registerR2(root *cobra.Command, globals shared.GlobalsFunc) {
	r2 := &cobra.Command{
		Use:   "r2",
		Short: "Read Cloudflare R2 buckets",
	}
	buckets := &cobra.Command{
		Use:   "buckets",
		Short: "Read R2 buckets",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List R2 buckets for an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				accountID, err := requireAccountID(resolved)
				if err != nil {
					return err
				}
				items, info, err := client.R2Buckets(ctx, accountID, nil)
				if err != nil {
					return err
				}
				return shared.WriteRawPaginatedList(items, info, flags.Format)
			})
		},
	}
	get := &cobra.Command{
		Use:   "get <bucket-name>",
		Short: "Get one R2 bucket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				accountID, err := requireAccountID(resolved)
				if err != nil {
					return err
				}
				raw, err := client.R2Bucket(ctx, accountID, args[0])
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, flags.Format)
				return nil
			})
		},
	}
	buckets.AddCommand(list, get)
	r2.AddCommand(buckets)
	root.AddCommand(r2)
}
