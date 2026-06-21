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
				return shared.WriteRawPaginatedList(items, info, flags.Format)
			})
		},
	}
	get := &cobra.Command{
		Use:   "get <script-name>...",
		Short: "Get Workers script operational metadata (one or more names)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			// Resolve accountID before the per-id loop so a missing account is a
			// command-level error (stderr + exit 1) rather than per-id @unresolved.
			resolved, err := shared.ResolveProfile(flags)
			if err != nil {
				return err
			}
			accountID, err := requireAccountID(resolved)
			if err != nil {
				return err
			}
			return shared.GetEntities(flags, args, func(ctx context.Context, client *api.Client, _ *shared.ResolvedProfile, id string) (any, error) {
				subdomain, err := client.WorkerSubdomain(ctx, accountID, id)
				if err != nil {
					return nil, err
				}
				versions, info, err := client.WorkerVersions(ctx, accountID, id, nil)
				if err != nil {
					return nil, err
				}
				decodedVersions, err := shared.RawItemsToAny(versions)
				if err != nil {
					return nil, err
				}
				decodedSubdomain, err := decodeRaw(subdomain)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"script_name": id,
					"subdomain":   decodedSubdomain,
					"versions":    decodedVersions,
					"pagination":  info,
				}, nil
			})
		},
	}
	workers.AddCommand(list, get)
	root.AddCommand(workers)
}
