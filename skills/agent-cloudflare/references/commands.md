# agent-cloudflare commands

Use this reference when the top-level skill tells you which path to take but you need exact command syntax.

## Setup And Profiles

Prefer `profiles`, not `auth`, in user-facing instructions. The CLI may keep `auth` as a hidden compatibility alias, but API tokens are tied to Cloudflare profiles that can discover account and zone context.

```bash
agent-cloudflare usage
agent-cloudflare config show
agent-cloudflare profiles add <profile> --form
agent-cloudflare profiles update <profile> --form
agent-cloudflare profiles list
agent-cloudflare profiles check
agent-cloudflare profiles discover <profile> --zone example.com
```

Use `--profile <profile>` when the user names a profile or when more than one profile exists. Use `--account-id` or `--zone-id` when the command can see multiple Cloudflare accounts or zones.

## Investigations

```bash
agent-cloudflare investigate usage
agent-cloudflare investigate zone-health example.com
agent-cloudflare investigate traffic-spike example.com --since 1h
agent-cloudflare investigate dns-change example.com
agent-cloudflare investigate ssl-breakage example.com
agent-cloudflare investigate waf-block example.com
agent-cloudflare investigate worker-error --account-id <account_id>
agent-cloudflare investigate cache-miss example.com
```

Use investigations for broad incident questions because they emit evidence rows and findings that are easier to synthesize for the user.

## Resource Reads

```bash
agent-cloudflare accounts list
agent-cloudflare zones list
agent-cloudflare zones get example.com
agent-cloudflare dns list example.com --type A
agent-cloudflare ssl status example.com
agent-cloudflare cache settings example.com
agent-cloudflare rulesets list example.com
agent-cloudflare waiting-rooms list example.com
agent-cloudflare workers list --account-id <account_id>
agent-cloudflare workers get <script_name> --account-id <account_id>
agent-cloudflare kv namespaces list --account-id <account_id>
agent-cloudflare r2 buckets list --account-id <account_id>
agent-cloudflare audit list --account-id <account_id>
agent-cloudflare analytics traffic example.com --since 1h
agent-cloudflare snapshot zone example.com
agent-cloudflare baseline check [zone-name-or-id] --file baseline.json
agent-cloudflare zone-settings get <setting-id>... [--zone <zone-name-or-id>]
agent-cloudflare waiting-rooms get <waiting-room-id>... [--zone <zone-name-or-id>]
```

## Raw API Reads

Use raw API reads for Cloudflare endpoints that do not yet have a first-class command. Keep queries narrow and prefer GET.

```bash
agent-cloudflare api get /zones --query name=example.com
agent-cloudflare api get /zones/<zone_id>/dns_records --query type=CNAME
```

## Guarded Mutations

Always dry-run first and summarize the planned operation before any confirmed write.

```bash
agent-cloudflare cache purge example.com --url https://example.com/a --dry-run
agent-cloudflare dns create example.com --type CNAME --name app --content target.example.com --dry-run
agent-cloudflare waiting-rooms update wr_... example.com --enabled --dry-run
```

Only run the same command with `--confirm` after the user explicitly approves the mutation.
