package auth

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	"github.com/shhac/agent-cloudflare/internal/config"
	"github.com/shhac/agent-cloudflare/internal/output"
)

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
