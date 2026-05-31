package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func registerDNS(root *cobra.Command, globals shared.GlobalsFunc) {
	var recordType, name, content string

	dns := &cobra.Command{
		Use:   "dns",
		Short: "Read DNS records for a zone",
	}
	list := &cobra.Command{
		Use:   "list [zone-name-or-id]",
		Short: "List DNS records",
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
				params := url.Values{}
				shared.AddString(params, "type", recordType)
				shared.AddString(params, "name", name)
				shared.AddString(params, "content", content)
				items, info, err := client.DNSRecords(ctx, zoneID, params)
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
	list.Flags().StringVar(&recordType, "type", "", "DNS record type, such as A, CNAME, TXT")
	list.Flags().StringVar(&name, "name", "", "Exact DNS record name")
	list.Flags().StringVar(&content, "content", "", "DNS record content")
	dns.AddCommand(list)
	root.AddCommand(dns)
}
