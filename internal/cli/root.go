package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/cli/auth"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	"github.com/shhac/agent-cloudflare/internal/config"
	"github.com/shhac/agent-cloudflare/internal/output"
)

func newRootCmd(version string) *cobra.Command {
	globals := &shared.GlobalFlags{}
	globalsFunc := func() *shared.GlobalFlags { return globals }
	root := &cobra.Command{
		Use:           "agent-cloudflare",
		Short:         "Cloudflare operations CLI for AI agents",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			applyConfiguredDefaults(cmd, globals)
		},
	}
	root.PersistentFlags().StringVarP(&globals.Profile, "profile", "p", "", "Cloudflare profile alias (or AGENT_CLOUDFLARE_PROFILE)")
	root.PersistentFlags().StringVar(&globals.AccountID, "account-id", "", "Cloudflare account ID override")
	root.PersistentFlags().StringVar(&globals.ZoneID, "zone-id", "", "Cloudflare zone ID override")
	root.PersistentFlags().StringVar(&globals.Zone, "zone", "", "Cloudflare zone name override")
	root.PersistentFlags().StringVar(&globals.APIToken, "api-token", "", "API token override; never printed or persisted")
	root.PersistentFlags().StringVar(&globals.BaseURL, "base-url", "", "Cloudflare API base URL override for tests")
	root.PersistentFlags().StringVarP(&globals.Format, "format", "f", "", "Output format: json, yaml, jsonl")
	root.PersistentFlags().IntVarP(&globals.Timeout, "timeout", "t", 0, "Request timeout in milliseconds")
	root.PersistentFlags().BoolVarP(&globals.Debug, "debug", "d", false, "Log redacted HTTP request records to stderr")
	_ = root.PersistentFlags().MarkHidden("base-url")

	registerUsageCommand(root)
	registerConfig(root)
	auth.Register(root, globalsFunc)
	registerAccounts(root, globalsFunc)
	registerZones(root, globalsFunc)
	registerDNS(root, globalsFunc)
	registerZoneSettings(root, globalsFunc)
	registerSSL(root, globalsFunc)
	registerCache(root, globalsFunc)
	registerRulesets(root, globalsFunc)
	registerWaitingRooms(root, globalsFunc)
	registerWorkers(root, globalsFunc)
	registerKV(root, globalsFunc)
	registerR2(root, globalsFunc)
	registerInvestigate(root, globalsFunc)
	registerRawAPI(root, globalsFunc)

	return root
}

func applyConfiguredDefaults(cmd *cobra.Command, globals *shared.GlobalFlags) {
	cfg := config.Read()
	if cfg.Defaults.TimeoutMS != nil && !cmd.Root().PersistentFlags().Changed("timeout") {
		globals.Timeout = *cfg.Defaults.TimeoutMS
	}
	if globals.BaseURL == "" {
		globals.BaseURL = os.Getenv("AGENT_CLOUDFLARE_BASE_URL")
	}
}

func Execute(version string) error {
	err := newRootCmd(version).Execute()
	if err != nil {
		_, _ = fmt.Fprintln(output.Stderr(), err)
	}
	return err
}
