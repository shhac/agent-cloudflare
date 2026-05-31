package cli

import (
	"context"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func registerWaitingRooms(root *cobra.Command, globals shared.GlobalsFunc) {
	var account bool

	waitingRooms := &cobra.Command{
		Use:   "waiting-rooms",
		Short: "Read Cloudflare Waiting Rooms",
	}
	list := &cobra.Command{
		Use:   "list [zone-name-or-id]",
		Short: "List zone or account Waiting Rooms",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				scope, scopeID, err := resolveWaitingRoomScope(ctx, client, resolved, account, args)
				if err != nil {
					return err
				}
				items, info, err := client.WaitingRooms(ctx, scope, scopeID, nil)
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
	list.Flags().BoolVar(&account, "account", false, "List account-level Waiting Rooms using --account-id or profile default account")

	get := &cobra.Command{
		Use:   "get <waiting-room-id> [zone-name-or-id]",
		Short: "Get one zone Waiting Room",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				zoneRef := resolved.ZoneID
				if len(args) > 1 {
					zoneRef = args[1]
				}
				zoneID, err := resolveZoneID(ctx, client, resolved, zoneRef)
				if err != nil {
					return err
				}
				raw, err := client.WaitingRoom(ctx, zoneID, args[0])
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, flags.Format)
				return nil
			})
		},
	}
	waitingRooms.AddCommand(list, get)
	registerWaitingRoomUpdate(waitingRooms, globals)
	root.AddCommand(waitingRooms)
}

func registerWaitingRoomUpdate(parent *cobra.Command, globals shared.GlobalsFunc) {
	var enabled bool
	var queueAll bool
	var dryRun bool
	var confirm bool

	cmd := &cobra.Command{
		Use:   "update <waiting-room-id> [zone-name-or-id]",
		Short: "Patch a Waiting Room with dry-run or explicit confirmation",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireMutationMode(dryRun, confirm); err != nil {
				return err
			}
			body := map[string]any{}
			if cmd.Flags().Changed("enabled") {
				body["enabled"] = enabled
			}
			if cmd.Flags().Changed("queue-all") {
				body["queue_all"] = queueAll
			}
			if len(body) == 0 {
				return agenterrors.New("no Waiting Room updates requested", agenterrors.FixableByAgent).
					WithHint("Use --enabled or --queue-all")
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
				path := "/zones/" + zoneID + "/waiting_rooms/" + args[0]
				if dryRun {
					writeDryRun(client, flags, http.MethodPatch, path, body)
					return nil
				}
				raw, err := client.UpdateWaitingRoom(ctx, zoneID, args[0], body)
				if err != nil {
					return err
				}
				decoded, err := decodeRaw(raw)
				if err != nil {
					return err
				}
				shared.WriteItem(mutationResult("waiting_room.update", decoded), flags.Format)
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Set Waiting Room enabled state")
	cmd.Flags().BoolVar(&queueAll, "queue-all", false, "Set queue_all state")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the request without sending it")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Send the mutation")
	parent.AddCommand(cmd)
}

func resolveWaitingRoomScope(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, account bool, args []string) (string, string, error) {
	if account {
		if resolved.AccountID == "" {
			return "", "", agenterrors.New("--account-id is required for account-scoped Waiting Rooms", agenterrors.FixableByAgent).
				WithHint("Pass --account-id or store one with 'agent-cloudflare profiles discover <profile>'")
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
