package cli

import (
	"os"

	"github.com/spf13/cobra"

	libcli "github.com/shhac/lib-agent-cli/cli"
	agentmcp "github.com/shhac/lib-agent-mcp"

	"github.com/shhac/agent-cloudflare/internal/cli/auth"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	"github.com/shhac/agent-cloudflare/internal/config"
	"github.com/shhac/agent-cloudflare/internal/credential"
	"github.com/shhac/agent-cloudflare/internal/output"
)

// NewRootCmd builds the agent-cloudflare root command with the family's shared
// scaffolding (NewRoot) plus the Cloudflare domain persistent flags.
func NewRootCmd(version string) *cobra.Command {
	globals := &shared.GlobalFlags{}
	globalsFunc := func() *shared.GlobalFlags { return globals }
	var root *cobra.Command
	root = libcli.NewRoot(libcli.Options{
		Use:           "agent-cloudflare",
		Short:         "Cloudflare operations CLI for AI agents",
		Version:       version,
		Globals:       &globals.Globals,
		DefaultFormat: output.FormatNDJSON,
		ConfigDefaults: func() {
			applyConfiguredDefaults(root, globals)
		},
		UnknownHint: "run 'agent-cloudflare usage' to see the available domains",
	})
	root.PersistentFlags().StringVarP(&globals.Profile, "profile", "p", "", "Cloudflare profile alias (or AGENT_CLOUDFLARE_PROFILE)")
	root.PersistentFlags().StringVar(&globals.AccountID, "account-id", "", "Cloudflare account ID override")
	root.PersistentFlags().StringVar(&globals.ZoneID, "zone-id", "", "Cloudflare zone ID override")
	root.PersistentFlags().StringVar(&globals.Zone, "zone", "", "Cloudflare zone name override")
	root.PersistentFlags().StringVar(&globals.APIToken, "api-token", "", "API token override; never printed or persisted")
	root.PersistentFlags().StringVar(&globals.BaseURL, "base-url", "", "Cloudflare API base URL override for tests")
	_ = root.PersistentFlags().MarkHidden("base-url")

	registerUsageCommand(root)
	registerConfig(root, globalsFunc)
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
	registerAudit(root, globalsFunc)
	registerAnalytics(root, globalsFunc)
	registerInvestigate(root, globalsFunc)
	registerSnapshot(root, globalsFunc)
	registerBaseline(root, globalsFunc)
	registerRawAPI(root, globalsFunc)

	// NewRoot installs the structured unknown-command handler on the root only.
	// Apply the same handling to every domain group (and nested subgroup) so an
	// unknown leaf — e.g. "dns bogus" or "kv namespaces bogus" — returns a
	// structured error listing that group's commands instead of cobra usage text.
	installGroupUnknownHandlers(root)

	// Expose the whole command tree as an MCP server (added last, so it reflects
	// the complete tree). --color/--expose are output-shaping, irrelevant to a
	// tool call, so hide them from the generated schemas.
	// Opt the agent-facing groups into the MCP tool surface: each becomes one
	// coarse tool that dispatches its subcommands (with a "help" verb), so the
	// surface is ~one-tool-per-group instead of one-per-leaf. Credential/config/
	// usage commands are deliberately left out — they aren't agent tasks.
	exposeGroups(root,
		"accounts", "analytics", "api", "audit", "baseline", "cache", "dns", "investigate", "kv", "r2", "rulesets", "snapshot", "ssl", "waiting-rooms", "workers", "zone-settings", "zones")

	// Store local-OAuth secrets under the family reverse-DNS service
	// (app.paulie.agent-cloudflare.mcp), separate from the API credentials.
	root.AddCommand(agentmcp.Command(root,
		agentmcp.WithHiddenFlags("color", "expose"),
		agentmcp.WithOAuthKeyringService(credential.MCPKeychainService()),
	))

	return root
}

// installGroupUnknownHandlers walks cmd's descendants and installs
// HandleUnknownCommand on each parent group (a command with subcommands and no
// Run/RunE of its own). cmd itself is left untouched so the root keeps its own
// UnknownHint.
func installGroupUnknownHandlers(cmd *cobra.Command) {
	for _, sub := range cmd.Commands() {
		if len(sub.Commands()) == 0 {
			continue
		}
		if sub.RunE == nil && sub.Run == nil {
			libcli.HandleUnknownCommand(sub, "run 'agent-cloudflare usage' to see the available domains")
		}
		installGroupUnknownHandlers(sub)
	}
}

func applyConfiguredDefaults(cmd *cobra.Command, globals *shared.GlobalFlags) {
	cfg := config.Read()
	if cfg.Defaults.TimeoutMS != nil && !cmd.Root().PersistentFlags().Changed("timeout") {
		globals.TimeoutMS = *cfg.Defaults.TimeoutMS
	}
	if globals.BaseURL == "" {
		globals.BaseURL = os.Getenv("AGENT_CLOUDFLARE_BASE_URL")
	}
}

// ExecuteArgs builds the root, runs it with the given args, and renders any
// bubbled error in the structured contract. It returns the error (unlike
// libcli.Run, which exits) so tests can assert on it. main uses libcli.Run.
func ExecuteArgs(version string, args []string) error {
	cmd := NewRootCmd(version)
	if args != nil {
		cmd.SetArgs(args)
	}
	err := cmd.Execute()
	if err != nil {
		output.WriteError(output.Stderr(), err)
	}
	return err
}

// exposeGroups opts the named top-level commands into the MCP tool surface.
// A name with no matching command is skipped silently — the list is a curation
// of agent-facing groups, not a registration check.
func exposeGroups(root *cobra.Command, names ...string) {
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	for _, c := range root.Commands() {
		if want[c.Name()] {
			agentmcp.Expose(c)
		}
	}
}
