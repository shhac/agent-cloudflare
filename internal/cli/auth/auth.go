package auth

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	"github.com/shhac/agent-cloudflare/internal/config"
	"github.com/shhac/agent-cloudflare/internal/credential"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
	"github.com/shhac/agent-cloudflare/internal/output"
)

var (
	credentialStore  = credential.Store
	credentialRemove = credential.Remove
)

func Register(root *cobra.Command, globals shared.GlobalsFunc) {
	profiles := &cobra.Command{
		Use:   "profiles",
		Short: "Manage Cloudflare credential profiles",
	}
	registerAdd(profiles)
	registerUpdate(profiles)
	registerCheck(profiles, globals)
	registerDiscover(profiles, globals)
	registerDefault(profiles)
	registerList(profiles)
	registerRemove(profiles)
	root.AddCommand(profiles)

	authAlias := &cobra.Command{
		Use:    "auth",
		Short:  "Manage Cloudflare credential profiles",
		Hidden: true,
	}
	registerAdd(authAlias)
	registerUpdate(authAlias)
	registerCheck(authAlias, globals)
	registerDiscover(authAlias, globals)
	registerDefault(authAlias)
	registerList(authAlias)
	registerRemove(authAlias)
	root.AddCommand(authAlias)
}

func registerAdd(parent *cobra.Command) {
	var apiToken, accountID, accountName, zoneID, zoneName string
	var form bool

	cmd := &cobra.Command{
		Use:   "add <profile>",
		Short: "Add a Cloudflare profile with a Keychain-stored API token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if form {
				filledToken, err := promptTokenViaDialog(cmd.Context(), alias, apiToken)
				if err != nil {
					output.WriteError(output.Stderr(), err)
					return nil
				}
				apiToken = filledToken
			}
			if !shared.RequireFlag("api-token", apiToken, "Provide --api-token <secret> or use --form") {
				return nil
			}
			storage, err := credentialStore(alias, apiToken)
			if err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
					WithHint("Use --form on a local graphical machine so the token can be stored in Keychain"))
				return nil
			}
			profile := config.Profile{
				AccountID:      accountID,
				AccountName:    accountName,
				DefaultZoneID:  zoneID,
				DefaultZone:    zoneName,
				CredentialType: credential.Type(apiToken),
			}
			if zoneName != "" && zoneID != "" {
				profile.Zones = map[string]string{zoneName: zoneID}
			}
			if err := config.StoreProfile(alias, profile); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
					WithHint("Check that the config directory is writable"))
				return nil
			}
			shared.WriteItem(map[string]any{
				"status":          "added",
				"profile":         alias,
				"storage":         storage,
				"account_id":      accountID,
				"account_name":    accountName,
				"default_zone_id": zoneID,
				"default_zone":    zoneName,
				"credential_type": profile.CredentialType,
			}, "")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiToken, "api-token", "", "Cloudflare API token (required unless --form is used)")
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for the API token via a native OS dialog (LLM never sees the input)")
	cmd.Flags().StringVar(&accountID, "account-id", "", "Default Cloudflare account ID for account-scoped commands")
	cmd.Flags().StringVar(&accountName, "account-name", "", "Non-secret account label for humans")
	cmd.Flags().StringVar(&zoneID, "zone-id", "", "Default Cloudflare zone ID for zone-scoped commands")
	cmd.Flags().StringVar(&zoneName, "zone", "", "Default zone name, such as example.com")
	parent.AddCommand(cmd)
}

func registerUpdate(parent *cobra.Command) {
	var apiToken, accountID, accountName, zoneID, zoneName string
	var clearAccount, clearZone, setDefault, form bool

	cmd := &cobra.Command{
		Use:   "update <profile>",
		Short: "Update a profile key or non-secret metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			keyRequested := form || cmd.Flags().Changed("api-token")
			if !keyRequested && !cmd.Flags().Changed("account-id") && !cmd.Flags().Changed("account-name") &&
				!cmd.Flags().Changed("zone-id") && !cmd.Flags().Changed("zone") && !clearAccount && !clearZone && !setDefault {
				output.WriteError(output.Stderr(), agenterrors.New("no profile updates requested", agenterrors.FixableByAgent).
					WithHint("Use --api-token, --form, --account-id, --account-name, --zone-id, --zone, --clear-account, --clear-zone, or --default"))
				return nil
			}
			if _, ok := config.Read().Profiles[alias]; !ok {
				output.WriteError(output.Stderr(), agenterrors.Newf(agenterrors.FixableByHuman, "profile %q is not configured", alias).
					WithHint("Run 'agent-cloudflare profiles list' to see profiles or 'agent-cloudflare profiles add "+alias+" --form' to create it"))
				return nil
			}
			credentialType := ""
			storage := ""
			if keyRequested {
				if form {
					filledToken, err := promptTokenViaDialog(cmd.Context(), alias, apiToken)
					if err != nil {
						output.WriteError(output.Stderr(), err)
						return nil
					}
					apiToken = filledToken
				}
				if !shared.RequireFlag("api-token", apiToken, "Provide --api-token <secret> or use --form") {
					return nil
				}
				var err error
				storage, err = credentialStore(alias, apiToken)
				if err != nil {
					output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
						WithHint("Use --form on a local graphical machine so the token can be stored in Keychain"))
					return nil
				}
				credentialType = credential.Type(apiToken)
			}
			if err := config.UpdateProfile(alias, func(profile config.Profile) config.Profile {
				if cmd.Flags().Changed("account-id") {
					profile.AccountID = accountID
				}
				if cmd.Flags().Changed("account-name") {
					profile.AccountName = accountName
				}
				if clearAccount {
					profile.AccountID = ""
					profile.AccountName = ""
				}
				if cmd.Flags().Changed("zone-id") {
					profile.DefaultZoneID = zoneID
				}
				if cmd.Flags().Changed("zone") {
					profile.DefaultZone = zoneName
				}
				if clearZone {
					profile.DefaultZoneID = ""
					profile.DefaultZone = ""
					profile.Zones = nil
				}
				if profile.DefaultZone != "" && profile.DefaultZoneID != "" {
					if profile.Zones == nil {
						profile.Zones = map[string]string{}
					}
					profile.Zones[profile.DefaultZone] = profile.DefaultZoneID
				}
				if credentialType != "" {
					profile.CredentialType = credentialType
				}
				return profile
			}); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
					WithHint("Run 'agent-cloudflare profiles list' to see configured profiles"))
				return nil
			}
			if setDefault {
				if err := config.SetDefault(alias); err != nil {
					output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
						WithHint("Run 'agent-cloudflare profiles list' to see configured profiles"))
					return nil
				}
			}
			cfg := config.Read()
			profile := cfg.Profiles[alias]
			item := map[string]any{
				"status":          "updated",
				"profile":         alias,
				"default":         cfg.DefaultProfile == alias,
				"account_id":      profile.AccountID,
				"account_name":    profile.AccountName,
				"default_zone_id": profile.DefaultZoneID,
				"default_zone":    profile.DefaultZone,
				"credential":      "keychain",
				"credential_type": profile.CredentialType,
			}
			if storage != "" {
				item["storage"] = storage
			}
			shared.WriteItem(item, "")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiToken, "api-token", "", "Replacement Cloudflare API token")
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for the replacement API token via a native OS dialog")
	cmd.Flags().StringVar(&accountID, "account-id", "", "Default Cloudflare account ID")
	cmd.Flags().StringVar(&accountName, "account-name", "", "Non-secret account label")
	cmd.Flags().BoolVar(&clearAccount, "clear-account", false, "Clear this profile's account defaults")
	cmd.Flags().StringVar(&zoneID, "zone-id", "", "Default Cloudflare zone ID")
	cmd.Flags().StringVar(&zoneName, "zone", "", "Default zone name")
	cmd.Flags().BoolVar(&clearZone, "clear-zone", false, "Clear this profile's zone defaults")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Make this the default profile")
	parent.AddCommand(cmd)
}

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

func chooseAccount(items []json.RawMessage, currentAccountID string) (id, name string, candidates int) {
	if currentAccountID != "" {
		return currentAccountID, "", len(items)
	}
	if len(items) != 1 {
		return "", "", len(items)
	}
	var account struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.Unmarshal(items[0], &account)
	return account.ID, account.Name, len(items)
}

func chooseZone(items []json.RawMessage, preferredZone string) (id, name string, zoneMap map[string]string) {
	zoneMap = map[string]string{}
	type zoneRecord struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	zones := []zoneRecord{}
	for _, item := range items {
		var zone zoneRecord
		if err := json.Unmarshal(item, &zone); err != nil {
			continue
		}
		if zone.ID != "" && zone.Name != "" {
			zoneMap[zone.Name] = zone.ID
			zones = append(zones, zone)
		}
	}
	if preferredZone != "" {
		if zoneID := zoneMap[preferredZone]; zoneID != "" {
			return zoneID, preferredZone, zoneMap
		}
	}
	if len(zones) == 1 {
		return zones[0].ID, zones[0].Name, zoneMap
	}
	return "", "", zoneMap
}

func registerDefault(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "default <profile>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.SetDefault(args[0]); err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}
			shared.WriteItem(map[string]any{"status": "default_set", "profile": args[0]}, "")
			return nil
		},
	}
	parent.AddCommand(cmd)
}

func registerList(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured profiles without exposing secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Read()
			profiles := make([]map[string]any, 0, len(cfg.Profiles))
			for alias, profile := range cfg.Profiles {
				profiles = append(profiles, map[string]any{
					"profile":         alias,
					"default":         alias == cfg.DefaultProfile,
					"account_id":      profile.AccountID,
					"account_name":    profile.AccountName,
					"default_zone_id": profile.DefaultZoneID,
					"default_zone":    profile.DefaultZone,
					"credential":      "keychain",
					"credential_type": profile.CredentialType,
				})
			}
			sort.Slice(profiles, func(i, j int) bool {
				return profiles[i]["profile"].(string) < profiles[j]["profile"].(string)
			})
			items := make([]any, len(profiles))
			for i, profile := range profiles {
				items[i] = profile
			}
			shared.WritePaginatedList(items, nil, "")
			return nil
		},
	}
	parent.AddCommand(cmd)
}

func registerRemove(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "remove <profile>",
		Short: "Remove a profile and its Keychain credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := credentialRemove(args[0]); err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}
			if err := config.RemoveProfile(args[0]); err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}
			shared.WriteItem(map[string]any{"status": "removed", "profile": args[0]}, "")
			return nil
		},
	}
	parent.AddCommand(cmd)
}

func accountListParams(accountID string) url.Values {
	params := url.Values{}
	if accountID != "" {
		params.Set("id", accountID)
	}
	return params
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
