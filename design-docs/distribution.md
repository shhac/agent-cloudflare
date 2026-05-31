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
