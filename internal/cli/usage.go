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
  agent-cloudflare baseline check example.com --file baseline.json
  agent-cloudflare zone-settings get <setting-id>... [--zone <zone-name-or-id>]
  agent-cloudflare waiting-rooms get <waiting-room-id>... [--zone <zone-name-or-id>]
  agent-cloudflare api get /zones --query name=example.com

Explicit mutations:
  agent-cloudflare cache purge example.com --url https://example.com/a --dry-run
  agent-cloudflare cache purge example.com --url https://example.com/a --confirm --reason "incident mitigation"
  agent-cloudflare dns create example.com --type CNAME --name app --content target.example.com --dry-run
  agent-cloudflare waiting-rooms update wr_... example.com --enabled --dry-run

Output:
  Lists default to NDJSON/jsonl.
  Get (single + multi): get <id>... accepts one or more ids and returns one result per id, in input order.
  Default output is NDJSON: one line per id — the record, or {"@unresolved":{"id","reason","fixable_by","hint"?}} for
  an id that couldn't be resolved (e.g. not found). --format json|yaml collapses to one {"data":[…],"@unresolved":[…]}
  envelope. A single get <id> is the one-element case (NDJSON by default; pass --format json for the object).
  Item-level misses stay on stdout, exit 0; only command-level failures (auth, network) go to stderr, exit 1.
  zone-settings get and waiting-rooms get scope their zone via --zone <zone-name-or-id> (not a trailing positional).
  api get stays single (raw escape hatch, not an entity get).
  Errors are JSON on stderr: {"error":"...","fixable_by":"agent"|"human"|"retry","hint"?:"...","retry_after_seconds"?:N} (hint and retry_after_seconds optional).

Secrets:
  Prefer --form for setup. Stored API tokens are read internally from Keychain and are never printed.
  The hidden auth command is accepted as an alias for profiles.
`)
			return nil
		},
	}
	root.AddCommand(cmd)
}
