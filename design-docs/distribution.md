# Distribution

Releases are manual and must follow `.agents/commands/release.md`.

Release tooling:

- GitHub Actions CI runs tests, vet, and builds on pushes and pull requests.
- `.goreleaser.yaml` is available for the manual release instructions.
- The release command remains the source of truth for tagging, GitHub release creation, and Homebrew tap updates.

Do not add automatic release-on-push workflows unless the release command is explicitly redesigned.

Install paths to document after first release:

```bash
brew install shhac/tap/agent-cloudflare
brew upgrade shhac/tap/agent-cloudflare
```

Skill installation is repository-local for now:

```text
skills/agent-cloudflare/SKILL.md
```

## Skill onboarding

`skills/agent-cloudflare/SKILL.md` should stay as the agent onboarding layer, not a full command manual. It should include:

- when to use the skill
- secret-safety rules
- the routing choice between setup, investigations, direct resource reads, raw API reads, and guarded mutations
- the minimal first commands
- the output/error contract at a glance

Detailed command syntax, scenario playbooks, and output/error interpretation live in `skills/agent-cloudflare/references/` and should be opened only when needed:

- `scenarios.md`: incident-shaped workflows
- `commands.md`: exact command syntax
- `output.md`: NDJSON, evidence rows, error hints, secret handling, and mutation approval rules

This keeps LLM onboarding short while preserving incremental disclosure for deeper tasks.
