package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func registerAudit(root *cobra.Command, globals shared.GlobalsFunc) {
	var actor, action, resourceID string

	audit := &cobra.Command{
		Use:   "audit",
		Short: "Read Cloudflare account audit logs",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List account audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				accountID, err := requireAccountID(resolved)
				if err != nil {
					return err
				}
				params := url.Values{}
				shared.AddString(params, "actor.email", actor)
				shared.AddString(params, "action.type", action)
				shared.AddString(params, "resource.id", resourceID)
				items, info, err := client.AuditLogs(ctx, accountID, params)
				if err != nil {
					return err
				}
				return shared.WriteRawPaginatedList(items, info, flags.Format)
			})
		},
	}
	list.Flags().StringVar(&actor, "actor", "", "Filter by actor email when supported")
	list.Flags().StringVar(&action, "action", "", "Filter by action type when supported")
	list.Flags().StringVar(&resourceID, "resource-id", "", "Filter by resource ID when supported")
	audit.AddCommand(list)
	root.AddCommand(audit)
}
