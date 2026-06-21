package cli

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/lib-agent-cli/creds"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func registerZones(root *cobra.Command, globals shared.GlobalsFunc) {
	var accountID, status string

	zones := &cobra.Command{
		Use:   "zones",
		Short: "Read Cloudflare zones",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List zones",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				params := url.Values{}
				shared.AddString(params, "account.id", creds.FirstNonEmpty(accountID, resolved.AccountID))
				shared.AddString(params, "status", status)
				items, info, err := client.Zones(ctx, params)
				if err != nil {
					return err
				}
				return shared.WriteRawPaginatedList(items, info, flags.Format)
			})
		},
	}
	list.Flags().StringVar(&accountID, "account-id", "", "Filter by account ID")
	list.Flags().StringVar(&status, "status", "", "Filter by zone status")

	get := &cobra.Command{
		Use:   "get <zone-name-or-id>...",
		Short: "Get one or more zones by ID or exact zone name",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.GetEntities(globals(), args, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, id string) (any, error) {
				zoneID, err := resolveZoneID(ctx, client, resolved, id)
				if err != nil {
					return nil, err
				}
				raw, err := client.Zone(ctx, zoneID)
				if err != nil {
					return nil, err
				}
				return decodeRaw(raw)
			})
		},
	}
	zones.AddCommand(list, get)
	root.AddCommand(zones)
}

func resolveZoneID(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, value string) (string, error) {
	if value == "" {
		if resolved.ZoneID != "" {
			return resolved.ZoneID, nil
		}
		return "", agenterrors.New("zone is required", agenterrors.FixableByAgent).
			WithHint("Pass a zone name/ID, set --zone-id, or store a default zone with 'agent-cloudflare profiles discover <profile> --zone <zone>'")
	}
	if looksLikeCloudflareID(value) {
		return value, nil
	}
	if resolved.Profile.Zones != nil {
		if zoneID := resolved.Profile.Zones[value]; zoneID != "" {
			return zoneID, nil
		}
	}
	params := url.Values{"name": []string{value}}
	if resolved.AccountID != "" {
		params.Set("account.id", resolved.AccountID)
	}
	items, _, err := client.Zones(ctx, params)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", agenterrors.Newf(agenterrors.FixableByAgent, "zone %q not found", value).
			WithHint("Run 'agent-cloudflare zones list' and use the returned zone id")
	}
	if len(items) > 1 {
		return "", agenterrors.Newf(agenterrors.FixableByAgent, "zone %q matched multiple zones", value).
			WithHint("Pass --account-id or use the exact zone id")
	}
	var zone struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(items[0], &zone); err != nil {
		return "", agenterrors.Wrap(err, agenterrors.FixableByAgent).
			WithHint("Cloudflare returned an unexpected zone shape; use --debug and retry")
	}
	if zone.ID == "" {
		return "", agenterrors.New("Cloudflare zone result did not include an id", agenterrors.FixableByRetry).
			WithHint("Retry the command, or use 'agent-cloudflare zones list --format json' to inspect the returned zone object")
	}
	return zone.ID, nil
}

func looksLikeCloudflareID(value string) bool {
	if len(value) != 32 {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'f') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
