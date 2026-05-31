package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func registerUsageCommand(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show LLM-oriented command examples",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), `agent-cloudflare usage

Setup:
  agent-cloudflare profiles add prod --form --account-id <acct_id>
  agent-cloudflare profiles discover prod --zone example.com
  agent-cloudflare profiles check prod
  agent-cloudflare profiles list

Read-only exploration:
  agent-cloudflare accounts list
  agent-cloudflare zones list --account-id <acct_id>
  agent-cloudflare zones get example.com
  agent-cloudflare dns list example.com --type A
  agent-cloudflare ssl status example.com
  agent-cloudflare cache settings example.com
  agent-cloudflare rulesets list example.com
  agent-cloudflare waiting-rooms list example.com
  agent-cloudflare workers list --account-id <acct_id>
  agent-cloudflare kv namespaces list --account-id <acct_id>
  agent-cloudflare r2 buckets list --account-id <acct_id>
  agent-cloudflare audit list --account-id <acct_id>
  agent-cloudflare analytics traffic example.com --since 1h
  agent-cloudflare investigate zone-health example.com
  agent-cloudflare investigate traffic-spike example.com --since 1h
  agent-cloudflare snapshot zone example.com
  agent-cloudflare api get /zones --query name=example.com

Explicit mutations:
  agent-cloudflare cache purge example.com --url https://example.com/a --dry-run
  agent-cloudflare cache purge example.com --url https://example.com/a --confirm --reason "incident mitigation"
  agent-cloudflare dns create example.com --type CNAME --name app --content target.example.com --dry-run
  agent-cloudflare waiting-rooms update wr_... example.com --enabled --dry-run

Output:
  Lists default to NDJSON/jsonl.
  Single resources default to JSON.
  Errors are JSON on stderr with error, fixable_by, and hint.

Secrets:
  Prefer --form for setup. Stored API tokens are read internally from Keychain and are never printed.
  The hidden auth command is accepted as an alias for profiles.
`)
			return nil
		},
	}
	root.AddCommand(cmd)
}
