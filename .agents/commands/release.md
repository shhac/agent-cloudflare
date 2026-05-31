---
description: Build, release, and publish to Homebrew
argument-hint: <patch|minor|major>
---

# Release

Perform a full release of the `agent-cloudflare` CLI: version bump, tag, build,
GitHub release, and Homebrew tap update.

## Arguments

- `$ARGUMENTS` - version bump type: `patch`, `minor`, or `major`

## Instructions

### Pre-flight

1. Confirm `$ARGUMENTS` is exactly `patch`, `minor`, or `major`. If not, stop and ask.
2. Confirm the working tree is clean:
   ```bash
   git status --short
   ```
   If there are changes, stop and ask.
3. Confirm the current branch is `main` and it is up to date with `origin/main`.
4. Run:
   ```bash
   make test
   go vet ./...
   ```
   If either fails, stop and fix.
5. Determine the current version from the latest git tag:
   ```bash
   current=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
   ```

### Step 1: Version bump, tag, and push

Calculate the next version:

```bash
current=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
IFS='.' read -r major minor patch <<< "$current"

case "$ARGUMENTS" in
  patch) patch=$((patch + 1)) ;;
  minor) minor=$((minor + 1)); patch=0 ;;
  major) major=$((major + 1)); minor=0; patch=0 ;;
  *) echo "expected patch, minor, or major"; exit 1 ;;
esac

new_version="${major}.${minor}.${patch}"
echo "Releasing v${new_version}"
```

Then tag and push:

```bash
git tag "v${new_version}"
git push origin main "v${new_version}"
```

### Step 2: Build

Preferred path:

```bash
goreleaser release --clean
```

If `goreleaser` is not installed, build manually:

```bash
rm -rf dist/
mkdir -p dist
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-cloudflare-darwin-arm64" ./cmd/agent-cloudflare
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-cloudflare-darwin-amd64" ./cmd/agent-cloudflare
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-cloudflare-linux-amd64" ./cmd/agent-cloudflare
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-cloudflare-linux-arm64" ./cmd/agent-cloudflare
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-cloudflare-windows-amd64.exe" ./cmd/agent-cloudflare

cd dist
for bin in agent-cloudflare-darwin-arm64 agent-cloudflare-darwin-amd64 agent-cloudflare-linux-amd64 agent-cloudflare-linux-arm64; do
  tar czf "${bin}.tar.gz" "$bin"
done
zip agent-cloudflare-windows-amd64.zip agent-cloudflare-windows-amd64.exe
shasum -a 256 *.tar.gz *.zip > checksums-sha256.txt
cd ..
```

Smoke-test the native binary:

```bash
./dist/agent-cloudflare-darwin-arm64 --version
./dist/agent-cloudflare-darwin-arm64 usage
```

### Step 3: Create GitHub release

If GoReleaser created the GitHub release, verify it and skip manual creation:

```bash
gh release view "v${new_version}"
```

Otherwise:

```bash
prev_tag=$(git tag --sort=-v:refname | head -2 | tail -1)
notes=$(git log --pretty=format:"- %s" "${prev_tag}..v${new_version}" --no-merges | grep -v "^- v[0-9]" || true)

gh release create "v${new_version}" dist/*.tar.gz dist/*.zip dist/checksums-sha256.txt \
  --title "v${new_version}" \
  --notes "$notes"
```

Verify:

```bash
gh release view "v${new_version}"
```

### Step 4: Update Homebrew tap

The Homebrew formula lives in `../homebrew-tap` relative to this repo root.

```bash
ls ../homebrew-tap/Formula/agent-cloudflare.rb
```

If it does not exist, create it by copying the pattern from
`../homebrew-tap/Formula/agent-sql.rb`, replacing:

- Class name: `AgentCloudflare`
- desc: `"Cloudflare operations CLI for AI agents"`
- homepage: `https://github.com/shhac/agent-cloudflare`
- all `agent-sql` references with `agent-cloudflare`
- version, URLs, and SHA256 values
- test assertions for `agent-cloudflare --version` and `agent-cloudflare usage`

Then commit and push the tap:

```bash
cd ../homebrew-tap
git status --short
git add Formula/agent-cloudflare.rb
git commit -m "agent-cloudflare ${new_version}"
git push
cd -
```

### Step 5: Report

Show the user:

- New version number
- GitHub release URL
- Homebrew tap commit, if applicable
- `brew install shhac/tap/agent-cloudflare`
- `brew upgrade shhac/tap/agent-cloudflare`
