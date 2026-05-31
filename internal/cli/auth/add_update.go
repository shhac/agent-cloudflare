package auth

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	"github.com/shhac/agent-cloudflare/internal/config"
	"github.com/shhac/agent-cloudflare/internal/credential"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
	"github.com/shhac/agent-cloudflare/internal/output"
)

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

type profileUpdateRequest struct {
	Alias        string
	APIToken     string
	AccountID    string
	AccountName  string
	ZoneID       string
	ZoneName     string
	ClearAccount bool
	ClearZone    bool
	SetDefault   bool
	KeyRequested bool

	accountIDChanged   bool
	accountNameChanged bool
	zoneIDChanged      bool
	zoneNameChanged    bool
}

func registerUpdate(parent *cobra.Command) {
	var apiToken, accountID, accountName, zoneID, zoneName string
	var clearAccount, clearZone, setDefault, form bool

	cmd := &cobra.Command{
		Use:   "update <profile>",
		Short: "Update a profile key or non-secret metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := profileUpdateRequest{
				Alias:              args[0],
				APIToken:           apiToken,
				AccountID:          accountID,
				AccountName:        accountName,
				ZoneID:             zoneID,
				ZoneName:           zoneName,
				ClearAccount:       clearAccount,
				ClearZone:          clearZone,
				SetDefault:         setDefault,
				KeyRequested:       form || cmd.Flags().Changed("api-token"),
				accountIDChanged:   cmd.Flags().Changed("account-id"),
				accountNameChanged: cmd.Flags().Changed("account-name"),
				zoneIDChanged:      cmd.Flags().Changed("zone-id"),
				zoneNameChanged:    cmd.Flags().Changed("zone"),
			}
			if !req.hasChanges() {
				output.WriteError(output.Stderr(), agenterrors.New("no profile updates requested", agenterrors.FixableByAgent).
					WithHint("Use --api-token, --form, --account-id, --account-name, --zone-id, --zone, --clear-account, --clear-zone, or --default"))
				return nil
			}
			if _, ok := config.Read().Profiles[req.Alias]; !ok {
				output.WriteError(output.Stderr(), agenterrors.Newf(agenterrors.FixableByHuman, "profile %q is not configured", req.Alias).
					WithHint("Run 'agent-cloudflare profiles list' to see profiles or 'agent-cloudflare profiles add "+req.Alias+" --form' to create it"))
				return nil
			}
			if form {
				filledToken, err := promptTokenViaDialog(cmd.Context(), req.Alias, req.APIToken)
				if err != nil {
					output.WriteError(output.Stderr(), err)
					return nil
				}
				req.APIToken = filledToken
			}
			runProfileUpdate(req)
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

func (r profileUpdateRequest) hasChanges() bool {
	return r.KeyRequested || r.accountIDChanged || r.accountNameChanged ||
		r.zoneIDChanged || r.zoneNameChanged || r.ClearAccount || r.ClearZone || r.SetDefault
}

func runProfileUpdate(req profileUpdateRequest) {
	credentialType := ""
	storage := ""
	if req.KeyRequested {
		if !shared.RequireFlag("api-token", req.APIToken, "Provide --api-token <secret> or use --form") {
			return
		}
		var err error
		storage, err = credentialStore(req.Alias, req.APIToken)
		if err != nil {
			output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
				WithHint("Use --form on a local graphical machine so the token can be stored in Keychain"))
			return
		}
		credentialType = credential.Type(req.APIToken)
	}
	if err := config.UpdateProfile(req.Alias, func(profile config.Profile) config.Profile {
		return applyProfileUpdate(profile, req, credentialType)
	}); err != nil {
		output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
			WithHint("Run 'agent-cloudflare profiles list' to see configured profiles"))
		return
	}
	if req.SetDefault {
		if err := config.SetDefault(req.Alias); err != nil {
			output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
				WithHint("Run 'agent-cloudflare profiles list' to see configured profiles"))
			return
		}
	}
	cfg := config.Read()
	profile := cfg.Profiles[req.Alias]
	item := profileUpdateOutput(req.Alias, profile, cfg.DefaultProfile == req.Alias, storage)
	shared.WriteItem(item, "")
}

func applyProfileUpdate(profile config.Profile, req profileUpdateRequest, credentialType string) config.Profile {
	if req.accountIDChanged {
		profile.AccountID = req.AccountID
	}
	if req.accountNameChanged {
		profile.AccountName = req.AccountName
	}
	if req.ClearAccount {
		profile.AccountID = ""
		profile.AccountName = ""
	}
	if req.zoneIDChanged {
		profile.DefaultZoneID = req.ZoneID
	}
	if req.zoneNameChanged {
		profile.DefaultZone = req.ZoneName
	}
	if req.ClearZone {
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
}

func profileUpdateOutput(alias string, profile config.Profile, isDefault bool, storage string) map[string]any {
	item := map[string]any{
		"status":          "updated",
		"profile":         alias,
		"default":         isDefault,
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
	return item
}
