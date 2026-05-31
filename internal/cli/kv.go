package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func registerKV(root *cobra.Command, globals shared.GlobalsFunc) {
	kv := &cobra.Command{
		Use:   "kv",
		Short: "Read Workers KV namespaces",
	}
	namespaces := &cobra.Command{
		Use:   "namespaces",
		Short: "Read Workers KV namespaces",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List KV namespaces for an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				accountID, err := requireAccountID(resolved)
				if err != nil {
					return err
				}
				items, info, err := client.KVNamespaces(ctx, accountID, nil)
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
		Use:   "get <namespace-id>",
		Short: "Get one KV namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				accountID, err := requireAccountID(resolved)
				if err != nil {
					return err
				}
				raw, err := client.KVNamespace(ctx, accountID, args[0])
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, flags.Format)
				return nil
			})
		},
	}
	namespaces.AddCommand(list, get)
	kv.AddCommand(namespaces)
	root.AddCommand(kv)
}
