package auth

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	"github.com/shhac/agent-cloudflare/internal/config"
	"github.com/shhac/agent-cloudflare/internal/credential"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
	"github.com/shhac/agent-cloudflare/internal/output"
)

func registerCheck(parent *cobra.Command, globals shared.GlobalsFunc) {
	cmd := &cobra.Command{
		Use:   "check [profile]",
		Short: "Verify stored credentials with /user/tokens/verify",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			if len(args) > 0 {
				flags.Profile = args[0]
			}
			resolved, err := shared.ResolveProfile(flags)
			if err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}
			credentialType := credential.Type(resolved.Token)
			if resolved.Alias != "override" && credentialType != resolved.Profile.CredentialType {
				_ = config.UpdateProfile(resolved.Alias, func(profile config.Profile) config.Profile {
					profile.CredentialType = credentialType
					return profile
				})
			}
			err = shared.WithResolvedClient(flags, resolved, func(ctx context.Context, client *api.Client) error {
				raw, err := client.VerifyToken(ctx)
				if err != nil {
					return err
				}
				var result any
				if err := json.Unmarshal(raw, &result); err != nil {
					return err
				}
				shared.WriteItem(map[string]any{
					"status":          "ok",
					"profile":         resolved.Alias,
					"credential_type": credentialType,
					"token":           result,
				}, "")
				return nil
			})
			if err != nil {
				output.WriteError(output.Stderr(), err)
			}
			return nil
		},
	}
	parent.AddCommand(cmd)
}

func registerDiscover(parent *cobra.Command, globals shared.GlobalsFunc) {
	var zoneName string

	cmd := &cobra.Command{
		Use:   "discover [profile]",
		Short: "Discover and store non-secret account and zone defaults",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			if len(args) > 0 {
				flags.Profile = args[0]
			}
			resolved, err := shared.ResolveProfile(flags)
			if err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}
			if resolved.Alias == "override" {
				output.WriteError(output.Stderr(), agenterrors.New("profiles discover requires a stored profile", agenterrors.FixableByHuman).
					WithHint("Run 'agent-cloudflare profiles add <profile> --form' first, then discover metadata for it"))
				return nil
			}
			err = shared.WithResolvedClient(flags, resolved, func(ctx context.Context, client *api.Client) error {
				accounts, _, err := client.Accounts(ctx, accountListParams(resolved.AccountID))
				if err != nil {
					return err
				}
				accountID, accountName, accountCandidates := chooseAccount(accounts, resolved.AccountID)
				zoneParams := url.Values{}
				if accountID != "" {
					zoneParams.Set("account.id", accountID)
				}
				if zoneName != "" {
					zoneParams.Set("name", zoneName)
				}
				zones, _, err := client.Zones(ctx, zoneParams)
				if err != nil {
					return err
				}
				defaultZoneID, defaultZoneName, zoneMap := chooseZone(zones, firstNonEmpty(zoneName, resolved.Zone))
				if err := config.UpdateProfile(resolved.Alias, func(profile config.Profile) config.Profile {
					if accountID != "" {
						profile.AccountID = accountID
					}
					if accountName != "" {
						profile.AccountName = accountName
					}
					if defaultZoneID != "" {
						profile.DefaultZoneID = defaultZoneID
					}
					if defaultZoneName != "" {
						profile.DefaultZone = defaultZoneName
					}
					if len(zoneMap) > 0 {
						profile.Zones = zoneMap
					}
					return profile
				}); err != nil {
					return agenterrors.Wrap(err, agenterrors.FixableByHuman).
						WithHint("Check that the config directory is writable and rerun profiles discover")
				}
				shared.WriteItem(map[string]any{
					"status":             "discovered",
					"profile":            resolved.Alias,
					"account_id":         accountID,
					"account_name":       accountName,
					"account_candidates": accountCandidates,
					"default_zone_id":    defaultZoneID,
					"default_zone":       defaultZoneName,
					"zones_discovered":   len(zoneMap),
				}, flags.Format)
				return nil
			})
			if err != nil {
				output.WriteError(output.Stderr(), err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&zoneName, "zone", "", "Prefer this exact zone as the default")
	parent.AddCommand(cmd)
}
