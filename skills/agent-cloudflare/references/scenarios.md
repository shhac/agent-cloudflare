# agent-cloudflare scenarios

## Site Down

Start broad:

```bash
agent-cloudflare investigate zone-health example.com
agent-cloudflare analytics traffic example.com --since 1h
agent-cloudflare audit list --account-id <account_id>
```

Look for inactive zone status, DNS gaps, SSL mode warnings, development mode, recent rule/DNS changes, and 5xx analytics.

## DNS Not Resolving

```bash
agent-cloudflare dns list example.com
agent-cloudflare investigate dns-change example.com
agent-cloudflare snapshot zone example.com
```

Use audit logs to connect recent record edits with symptoms.

## SSL Breakage

```bash
agent-cloudflare ssl status example.com
agent-cloudflare investigate ssl-breakage example.com
```

Look for SSL mode weaker than Full (strict), Always Use HTTPS off, old TLS minimums, and automatic HTTPS rewrite state.

## Users Blocked Or Challenged

```bash
agent-cloudflare rulesets list example.com
agent-cloudflare investigate waf-block example.com
agent-cloudflare analytics traffic example.com --since 1h
```

Use ruleset summaries first, then raw API paths if a specific ruleset or security event dataset is needed.

## Worker Errors

```bash
agent-cloudflare workers list --account-id <account_id>
agent-cloudflare workers get <script_name> --account-id <account_id>
agent-cloudflare investigate worker-error --account-id <account_id>
```

The current CLI surfaces inventory and versions. Deeper Worker metrics should be added through analytics datasets.

## Cache Miss Or Origin Load

```bash
agent-cloudflare cache settings example.com
agent-cloudflare investigate cache-miss example.com
agent-cloudflare analytics traffic example.com --since 1h
```

Check cache settings and GraphQL cache status distribution before considering a cache purge.

## Mutation Preview

Always dry-run first:

```bash
agent-cloudflare cache purge example.com --url https://example.com/a --dry-run
agent-cloudflare dns create example.com --type CNAME --name app --content target.example.com --dry-run
agent-cloudflare waiting-rooms update wr_... example.com --enabled --dry-run
```

Only use `--confirm` after the user explicitly approves the mutation.
