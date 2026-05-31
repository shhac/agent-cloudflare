package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func registerWorkers(root *cobra.Command, globals shared.GlobalsFunc) {
	workers := &cobra.Command{
		Use:   "workers",
		Short: "Read Cloudflare Workers scripts",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List Workers scripts for an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				accountID, err := requireAccountID(resolved)
				if err != nil {
					return err
				}
				items, info, err := client.Workers(ctx, accountID, nil)
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
	get := &cobra.Command{
		Use:   "get <script-name>",
		Short: "Get Workers script operational metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				accountID, err := requireAccountID(resolved)
				if err != nil {
					return err
				}
				subdomain, err := client.WorkerSubdomain(ctx, accountID, args[0])
				if err != nil {
					return err
				}
				versions, info, err := client.WorkerVersions(ctx, accountID, args[0], nil)
				if err != nil {
					return err
				}
				decodedVersions, err := shared.RawItemsToAny(versions)
				if err != nil {
					return err
				}
				decodedSubdomain, err := decodeRaw(subdomain)
				if err != nil {
					return err
				}
				shared.WriteItem(map[string]any{
					"script_name": args[0],
					"subdomain":   decodedSubdomain,
					"versions":    decodedVersions,
					"pagination":  info,
				}, flags.Format)
				return nil
			})
		},
	}
	workers.AddCommand(list, get)
	root.AddCommand(workers)
}
