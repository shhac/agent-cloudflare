package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
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
				return shared.WriteRawPaginatedList(items, info, flags.Format)
			})
		},
	}
	list.Flags().StringVar(&recordType, "type", "", "DNS record type, such as A, CNAME, TXT")
	list.Flags().StringVar(&name, "name", "", "Exact DNS record name")
	list.Flags().StringVar(&content, "content", "", "DNS record content")
	dns.AddCommand(list)
	registerDNSCreate(dns, globals)
	registerDNSUpdate(dns, globals)
	root.AddCommand(dns)
}

func registerDNSCreate(parent *cobra.Command, globals shared.GlobalsFunc) {
	var recordType, name, content string
	var proxied bool
	var ttl int
	var dryRun, confirm bool

	cmd := &cobra.Command{
		Use:   "create [zone-name-or-id]",
		Short: "Create a DNS record with dry-run or explicit confirmation",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireMutationMode(dryRun, confirm); err != nil {
				return err
			}
			if recordType == "" || name == "" || content == "" {
				return agenterrors.New("--type, --name, and --content are required", agenterrors.FixableByAgent).
					WithHint("Example: agent-cloudflare dns create example.com --type CNAME --name app --content target.example.com --dry-run")
			}
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				zoneID, err := zoneIDFromArgs(ctx, client, resolved, args)
				if err != nil {
					return err
				}
				body := map[string]any{
					"type":    recordType,
					"name":    name,
					"content": content,
				}
				if cmd.Flags().Changed("proxied") {
					body["proxied"] = proxied
				}
				if ttl > 0 {
					body["ttl"] = ttl
				}
				path := "/zones/" + zoneID + "/dns_records"
				return executeMutation(ctx, client, flags, mutationRequest{
					DryRun:  dryRun,
					Confirm: confirm,
					Method:  http.MethodPost,
					Path:    path,
					Body:    body,
					Action:  "dns.create",
					Send: func(ctx context.Context) (json.RawMessage, error) {
						return client.CreateDNSRecord(ctx, zoneID, body)
					},
				})
			})
		},
	}
	cmd.Flags().StringVar(&recordType, "type", "", "DNS record type, such as A, CNAME, TXT")
	cmd.Flags().StringVar(&name, "name", "", "DNS record name")
	cmd.Flags().StringVar(&content, "content", "", "DNS record content")
	cmd.Flags().BoolVar(&proxied, "proxied", false, "Whether Cloudflare proxying should be enabled")
	cmd.Flags().IntVar(&ttl, "ttl", 0, "DNS TTL; omit for Cloudflare default")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the request without sending it")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Send the mutation")
	parent.AddCommand(cmd)
}

func registerDNSUpdate(parent *cobra.Command, globals shared.GlobalsFunc) {
	var recordType, name, content string
	var proxied bool
	var ttl int
	var dryRun, confirm bool

	cmd := &cobra.Command{
		Use:   "update <record-id> [zone-name-or-id]",
		Short: "Patch a DNS record with dry-run or explicit confirmation",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireMutationMode(dryRun, confirm); err != nil {
				return err
			}
			body := map[string]any{}
			if cmd.Flags().Changed("type") {
				body["type"] = recordType
			}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("content") {
				body["content"] = content
			}
			if cmd.Flags().Changed("proxied") {
				body["proxied"] = proxied
			}
			if cmd.Flags().Changed("ttl") {
				body["ttl"] = ttl
			}
			if len(body) == 0 {
				return agenterrors.New("no DNS updates requested", agenterrors.FixableByAgent).
					WithHint("Use --type, --name, --content, --proxied, or --ttl")
			}
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				zoneArgs := []string{}
				if len(args) > 1 {
					zoneArgs = []string{args[1]}
				}
				zoneID, err := zoneIDFromArgs(ctx, client, resolved, zoneArgs)
				if err != nil {
					return err
				}
				path := "/zones/" + zoneID + "/dns_records/" + args[0]
				return executeMutation(ctx, client, flags, mutationRequest{
					DryRun:  dryRun,
					Confirm: confirm,
					Method:  http.MethodPatch,
					Path:    path,
					Body:    body,
					Action:  "dns.update",
					Send: func(ctx context.Context) (json.RawMessage, error) {
						return client.UpdateDNSRecord(ctx, zoneID, args[0], body)
					},
				})
			})
		},
	}
	cmd.Flags().StringVar(&recordType, "type", "", "DNS record type")
	cmd.Flags().StringVar(&name, "name", "", "DNS record name")
	cmd.Flags().StringVar(&content, "content", "", "DNS record content")
	cmd.Flags().BoolVar(&proxied, "proxied", false, "Whether Cloudflare proxying should be enabled")
	cmd.Flags().IntVar(&ttl, "ttl", 0, "DNS TTL")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the request without sending it")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Send the mutation")
	parent.AddCommand(cmd)
}

func zoneIDFromArgs(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, args []string) (string, error) {
	zoneRef := resolved.ZoneID
	if len(args) > 0 {
		zoneRef = args[0]
	}
	return resolveZoneID(ctx, client, resolved, zoneRef)
}
