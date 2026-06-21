package cli

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shhac/agent-cloudflare/internal/config"
	"github.com/shhac/agent-cloudflare/internal/credential"
	"github.com/shhac/agent-cloudflare/internal/mockcloudflare"
	"github.com/shhac/agent-cloudflare/internal/output"
)

type commandResult struct {
	stdout string
	stderr string
	err    error
}

type memoryBackend struct {
	tokens  map[string]string
	deleted []string
}

func (m *memoryBackend) Store(name, token string) error {
	if m.tokens == nil {
		m.tokens = map[string]string{}
	}
	m.tokens[name] = token
	return nil
}

func (m *memoryBackend) Get(name string) (string, error) {
	return m.tokens[name], nil
}

func (m *memoryBackend) Delete(name string) {
	m.deleted = append(m.deleted, name)
	delete(m.tokens, name)
}

func runCommand(t *testing.T, args ...string) commandResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	restoreWriters := output.SetWriters(&stdout, &stderr)
	t.Cleanup(restoreWriters)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func runExecute(t *testing.T, args ...string) commandResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	restoreWriters := output.SetWriters(&stdout, &stderr)
	t.Cleanup(restoreWriters)

	err := ExecuteArgs("test", args)
	return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func withMockServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(mockcloudflare.NewServer())
	t.Cleanup(server.Close)
	return server.URL
}

func withTempConfigAndKeychain(t *testing.T) *memoryBackend {
	t.Helper()
	config.SetConfigDir(t.TempDir())
	t.Cleanup(func() { config.SetConfigDir("") })
	backend := &memoryBackend{tokens: map[string]string{}}
	restoreBackend := credential.SetBackendForTest(backend)
	t.Cleanup(restoreBackend)
	return backend
}

func TestHelpHidesAuthAlias(t *testing.T) {
	result := runCommand(t, "--help")
	if result.err != nil {
		t.Fatalf("Execute() error = %v", result.err)
	}
	if strings.Contains(result.stdout, "auth") {
		t.Fatalf("help should hide auth alias:\n%s", result.stdout)
	}
	if !strings.Contains(result.stdout, "profiles") {
		t.Fatalf("help should include profiles:\n%s", result.stdout)
	}
}

func TestCompletionCommandAvailable(t *testing.T) {
	zsh := runCommand(t, "completion", "zsh")
	if zsh.err != nil || zsh.stderr != "" {
		t.Fatalf("completion zsh err=%v stderr=%s stdout=%s", zsh.err, zsh.stderr, zsh.stdout)
	}
	if !strings.Contains(zsh.stdout, "#compdef agent-cloudflare") {
		t.Fatalf("completion zsh stdout = %s, want compdef", zsh.stdout)
	}

	bash := runCommand(t, "completion", "bash")
	if bash.err != nil || bash.stderr != "" {
		t.Fatalf("completion bash err=%v stderr=%s stdout=%s", bash.err, bash.stderr, bash.stdout)
	}
	if !strings.Contains(bash.stdout, "bash completion") {
		t.Fatalf("completion bash stdout = %s, want bash completion script", bash.stdout)
	}
}

func TestHiddenAuthAliasWorks(t *testing.T) {
	baseURL := withMockServer(t)
	result := runCommand(t, "--base-url", baseURL, "--api-token", "cfut_mock", "auth", "check")
	if result.err != nil {
		t.Fatalf("Execute() error = %v", result.err)
	}
	if result.stderr != "" {
		t.Fatalf("stderr = %s", result.stderr)
	}
	if !strings.Contains(result.stdout, `"status": "ok"`) {
		t.Fatalf("stdout = %s, want ok status", result.stdout)
	}
}

func TestProfileResolutionUsesStoredTokenAndZone(t *testing.T) {
	backend := withTempConfigAndKeychain(t)
	baseURL := withMockServer(t)

	add := runCommand(t, "--base-url", baseURL, "profiles", "add", "prod", "--api-token", "cfut_mock", "--zone-id", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "--account-id", "023e105f4ecef8ad9ca31a8372d0c353")
	if add.err != nil || add.stderr != "" {
		t.Fatalf("profiles add err=%v stderr=%s stdout=%s", add.err, add.stderr, add.stdout)
	}
	if backend.tokens["prod"] != "cfut_mock" {
		t.Fatalf("stored token = %q", backend.tokens["prod"])
	}

	list := runCommand(t, "--base-url", baseURL, "--profile", "prod", "dns", "list")
	if list.err != nil || list.stderr != "" {
		t.Fatalf("dns list err=%v stderr=%s stdout=%s", list.err, list.stderr, list.stdout)
	}
	if !strings.Contains(list.stdout, `"dns_mock_a"`) || !strings.Contains(list.stdout, `"@pagination"`) {
		t.Fatalf("stdout = %s, want DNS NDJSON with pagination", list.stdout)
	}
}

func TestProfilesDiscoverStoresAccountAndZoneMetadata(t *testing.T) {
	_ = withTempConfigAndKeychain(t)
	baseURL := withMockServer(t)
	add := runCommand(t, "--base-url", baseURL, "profiles", "add", "prod", "--api-token", "cfut_mock")
	if add.err != nil || add.stderr != "" {
		t.Fatalf("profiles add err=%v stderr=%s stdout=%s", add.err, add.stderr, add.stdout)
	}

	result := runCommand(t, "--base-url", baseURL, "--profile", "prod", "profiles", "discover", "--zone", "example.com")
	if result.err != nil || result.stderr != "" {
		t.Fatalf("profiles discover err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
	}
	cfg := config.Read()
	profile := cfg.Profiles["prod"]
	if profile.DefaultZone != "example.com" || profile.DefaultZoneID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("profile zone = %#v", profile)
	}
	if profile.Zones["example.com"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("profile zones = %#v", profile.Zones)
	}
}

func TestProfileLifecycleDoesNotExposeSecrets(t *testing.T) {
	backend := withTempConfigAndKeychain(t)
	baseURL := withMockServer(t)
	secrets := []string{"cfut_prod_secret", "v1.0-dev-secret", "cfut_replacement_secret"}

	addProd := runCommand(t, "--base-url", baseURL, "profiles", "add", "prod", "--api-token", secrets[0], "--account-id", "acct_prod")
	if addProd.err != nil || addProd.stderr != "" {
		t.Fatalf("profiles add prod err=%v stderr=%s stdout=%s", addProd.err, addProd.stderr, addProd.stdout)
	}
	addDev := runCommand(t, "--base-url", baseURL, "profiles", "add", "dev", "--api-token", secrets[1])
	if addDev.err != nil || addDev.stderr != "" {
		t.Fatalf("profiles add dev err=%v stderr=%s stdout=%s", addDev.err, addDev.stderr, addDev.stdout)
	}
	update := runCommand(t, "--base-url", baseURL, "profiles", "update", "prod", "--api-token", secrets[2], "--account-name", "Production", "--zone-id", "zone_prod", "--zone", "example.org", "--default")
	if update.err != nil || update.stderr != "" {
		t.Fatalf("profiles update err=%v stderr=%s stdout=%s", update.err, update.stderr, update.stdout)
	}
	if backend.tokens["prod"] != secrets[2] {
		t.Fatalf("updated token = %q", backend.tokens["prod"])
	}
	cfg := config.Read()
	prod := cfg.Profiles["prod"]
	if cfg.DefaultProfile != "prod" || prod.AccountID != "acct_prod" || prod.AccountName != "Production" ||
		prod.DefaultZoneID != "zone_prod" || prod.DefaultZone != "example.org" || prod.Zones["example.org"] != "zone_prod" {
		t.Fatalf("profile config = %#v default=%q", prod, cfg.DefaultProfile)
	}

	list := runCommand(t, "--base-url", baseURL, "profiles", "list")
	if list.err != nil || list.stderr != "" {
		t.Fatalf("profiles list err=%v stderr=%s stdout=%s", list.err, list.stderr, list.stdout)
	}
	if !strings.Contains(list.stdout, `"profile":"dev"`) || !strings.Contains(list.stdout, `"profile":"prod"`) {
		t.Fatalf("profiles list stdout = %s", list.stdout)
	}
	if strings.Index(list.stdout, `"profile":"dev"`) > strings.Index(list.stdout, `"profile":"prod"`) {
		t.Fatalf("profiles list should be sorted: %s", list.stdout)
	}

	remove := runCommand(t, "--base-url", baseURL, "profiles", "remove", "dev")
	if remove.err != nil || remove.stderr != "" {
		t.Fatalf("profiles remove err=%v stderr=%s stdout=%s", remove.err, remove.stderr, remove.stdout)
	}
	if len(backend.deleted) != 1 || backend.deleted[0] != "dev" {
		t.Fatalf("deleted = %#v", backend.deleted)
	}
	if _, ok := config.Read().Profiles["dev"]; ok {
		t.Fatalf("dev profile still configured")
	}

	configBytes, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	allVisible := addProd.stdout + addProd.stderr + addDev.stdout + addDev.stderr + update.stdout + update.stderr + list.stdout + list.stderr + remove.stdout + remove.stderr + string(configBytes)
	for _, secret := range secrets {
		if strings.Contains(allVisible, secret) {
			t.Fatalf("secret %q leaked in visible output/config:\n%s", secret, allVisible)
		}
	}
}

func TestConfigCommandsHonorFormatFlag(t *testing.T) {
	_ = withTempConfigAndKeychain(t)

	// Default for single items stays pretty JSON.
	def := runCommand(t, "config", "path")
	if def.err != nil || def.stderr != "" {
		t.Fatalf("config path err=%v stderr=%s stdout=%s", def.err, def.stderr, def.stdout)
	}
	if !strings.HasPrefix(strings.TrimSpace(def.stdout), "{") {
		t.Fatalf("config path default = %s, want JSON object", def.stdout)
	}

	// --format yaml must reach config output rather than silently falling back.
	y := runCommand(t, "config", "path", "--format", "yaml")
	if y.err != nil || y.stderr != "" {
		t.Fatalf("config path yaml err=%v stderr=%s stdout=%s", y.err, y.stderr, y.stdout)
	}
	if !strings.HasPrefix(strings.TrimSpace(y.stdout), "path:") {
		t.Fatalf("config path --format yaml = %s, want YAML", y.stdout)
	}
}

func TestAccountResourceCommands(t *testing.T) {
	baseURL := withMockServer(t)
	baseArgs := []string{"--base-url", baseURL, "--api-token", "cfut_mock", "--account-id", "023e105f4ecef8ad9ca31a8372d0c353"}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "workers list", args: []string{"workers", "list"}, want: "api-worker"},
		{name: "workers get", args: []string{"workers", "get", "api-worker"}, want: "worker_version_2"},
		{name: "kv namespaces list", args: []string{"kv", "namespaces", "list"}, want: "kv_mock_sessions"},
		{name: "kv namespaces get", args: []string{"kv", "namespaces", "get", "kv_mock_sessions"}, want: "sessions"},
		{name: "r2 buckets list", args: []string{"r2", "buckets", "list"}, want: "assets"},
		{name: "r2 buckets get", args: []string{"r2", "buckets", "get", "assets"}, want: "Standard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append(append([]string{}, baseArgs...), tt.args...)
			result := runCommand(t, args...)
			if result.err != nil || result.stderr != "" {
				t.Fatalf("command err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
			}
			if !strings.Contains(result.stdout, tt.want) {
				t.Fatalf("stdout = %s, want %s", result.stdout, tt.want)
			}
		})
	}
}

func TestMissingZoneReturnsJSONErrorHint(t *testing.T) {
	baseURL := withMockServer(t)
	result := runExecute(t, "--base-url", baseURL, "--api-token", "cfut_mock", "dns", "list")
	if result.err == nil {
		t.Fatalf("expected command error when no zone is resolvable")
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %s, want empty", result.stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.stderr), &payload); err != nil {
		t.Fatalf("stderr is not JSON: %s", result.stderr)
	}
	if payload["fixable_by"] != "agent" || payload["hint"] == "" {
		t.Fatalf("payload = %#v, want agent error with hint", payload)
	}
}

func TestAccountScopeErrorHasDiscoveryHint(t *testing.T) {
	baseURL := withMockServer(t)
	result := runExecute(t, "--base-url", baseURL, "--api-token", "cfut_mock", "workers", "list")
	if result.err == nil {
		t.Fatalf("expected command error when no account is resolvable")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.stderr), &payload); err != nil {
		t.Fatalf("stderr is not JSON: %s", result.stderr)
	}
	if payload["fixable_by"] != "agent" || !strings.Contains(payload["hint"].(string), "profiles discover") {
		t.Fatalf("payload = %#v, want profiles discover hint", payload)
	}
}

func TestZoneHealthInvestigationEmitsEvidence(t *testing.T) {
	baseURL := withMockServer(t)
	result := runCommand(t, "--base-url", baseURL, "--api-token", "cfut_mock", "investigate", "zone-health", "example.com")
	if result.err != nil || result.stderr != "" {
		t.Fatalf("zone-health err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
	}
	for _, want := range []string{`"object":"zone"`, `"object":"ssl_settings"`, `"object":"cache_settings"`, `"object":"rulesets_summary"`, `"severity":"warning"`} {
		if !strings.Contains(result.stdout, want) {
			t.Fatalf("stdout missing %s:\n%s", want, result.stdout)
		}
	}
}

func TestAnalyticsAuditAndIncidentCommands(t *testing.T) {
	baseURL := withMockServer(t)
	baseArgs := []string{"--base-url", baseURL, "--api-token", "cfut_mock", "--account-id", "023e105f4ecef8ad9ca31a8372d0c353"}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "analytics traffic", args: []string{"analytics", "traffic", "example.com"}, want: "example.com"},
		{name: "audit list", args: []string{"audit", "list"}, want: "audit_dns_update"},
		{name: "traffic spike", args: []string{"investigate", "traffic-spike", "example.com"}, want: "traffic_analytics"},
		{name: "dns change", args: []string{"investigate", "dns-change", "example.com"}, want: "audit_logs"},
		{name: "ssl breakage", args: []string{"investigate", "ssl-breakage", "example.com"}, want: "ssl_settings"},
		{name: "waf block", args: []string{"investigate", "waf-block", "example.com"}, want: "rulesets_summary"},
		{name: "worker error", args: []string{"investigate", "worker-error"}, want: "workers_summary"},
		{name: "cache miss", args: []string{"investigate", "cache-miss", "example.com"}, want: "cache_settings"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append(append([]string{}, baseArgs...), tt.args...)
			result := runCommand(t, args...)
			if result.err != nil || result.stderr != "" {
				t.Fatalf("command err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
			}
			if !strings.Contains(result.stdout, tt.want) {
				t.Fatalf("stdout = %s, want %s", result.stdout, tt.want)
			}
		})
	}
}

func TestSnapshotAndBaselineCommands(t *testing.T) {
	baseURL := withMockServer(t)
	result := runCommand(t, "--base-url", baseURL, "--api-token", "cfut_mock", "snapshot", "zone", "example.com")
	if result.err != nil || result.stderr != "" {
		t.Fatalf("snapshot zone err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
	}
	if !strings.Contains(result.stdout, `"schema": "agent-cloudflare.zone-snapshot.v1"`) {
		t.Fatalf("stdout = %s, want snapshot schema", result.stdout)
	}

	dir := t.TempDir()
	before := filepath.Join(dir, "before.json")
	after := filepath.Join(dir, "after.json")
	if err := os.WriteFile(before, []byte(`{"schema":"agent-cloudflare.zone-snapshot.v1","zone":{"name":"example.com"}}`), 0o644); err != nil {
		t.Fatalf("WriteFile(before) error = %v", err)
	}
	if err := os.WriteFile(after, []byte(`{"schema":"agent-cloudflare.zone-snapshot.v1","zone":{"name":"example.net"}}`), 0o644); err != nil {
		t.Fatalf("WriteFile(after) error = %v", err)
	}
	diff := runCommand(t, "snapshot", "diff", before, after)
	if diff.err != nil || diff.stderr != "" {
		t.Fatalf("snapshot diff err=%v stderr=%s stdout=%s", diff.err, diff.stderr, diff.stdout)
	}
	if !strings.Contains(diff.stdout, `"path": "zone"`) {
		t.Fatalf("stdout = %s, want zone diff", diff.stdout)
	}
}

func TestEntityGetSmoke(t *testing.T) {
	baseURL := withMockServer(t)
	baseArgs := []string{"--base-url", baseURL, "--api-token", "cfut_mock"}

	t.Run("zones get single NDJSON", func(t *testing.T) {
		result := runCommand(t, append(baseArgs, "zones", "get", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")...)
		if result.err != nil || result.stderr != "" {
			t.Fatalf("err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
		}
		if !strings.Contains(result.stdout, `"example.com"`) {
			t.Fatalf("stdout = %s, want example.com", result.stdout)
		}
	})

	t.Run("zones get multi NDJSON two records", func(t *testing.T) {
		result := runCommand(t, append(baseArgs, "zones", "get", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")...)
		if result.err != nil || result.stderr != "" {
			t.Fatalf("err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
		}
		lines := strings.Split(strings.TrimSpace(result.stdout), "\n")
		if len(lines) != 2 {
			t.Fatalf("want 2 NDJSON lines, got %d: %s", len(lines), result.stdout)
		}
	})

	t.Run("zones get multi with miss emits @unresolved", func(t *testing.T) {
		result := runCommand(t, append(baseArgs, "zones", "get", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "ffffffffffffffffffffffffffffffff")...)
		// command-level exit 0, bad id → @unresolved on stdout
		if result.err != nil {
			t.Fatalf("unexpected err=%v stderr=%s", result.err, result.stderr)
		}
		if !strings.Contains(result.stdout, `"@unresolved"`) {
			t.Fatalf("stdout = %s, want @unresolved for missing zone", result.stdout)
		}
		if !strings.Contains(result.stdout, `"example.com"`) {
			t.Fatalf("stdout = %s, want first zone still present", result.stdout)
		}
	})

	t.Run("zone-settings get with --zone flag", func(t *testing.T) {
		result := runCommand(t, append(baseArgs, "zone-settings", "get", "ssl", "--zone", "example.com")...)
		if result.err != nil || result.stderr != "" {
			t.Fatalf("err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
		}
		if !strings.Contains(result.stdout, `"ssl"`) {
			t.Fatalf("stdout = %s, want ssl setting", result.stdout)
		}
	})

	t.Run("zone-settings get multi with --zone", func(t *testing.T) {
		result := runCommand(t, append(baseArgs, "zone-settings", "get", "ssl", "brotli", "--zone", "example.com")...)
		if result.err != nil || result.stderr != "" {
			t.Fatalf("err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
		}
		lines := strings.Split(strings.TrimSpace(result.stdout), "\n")
		if len(lines) != 2 {
			t.Fatalf("want 2 NDJSON lines, got %d: %s", len(lines), result.stdout)
		}
	})

	t.Run("zone-settings get missing setting emits @unresolved", func(t *testing.T) {
		result := runCommand(t, append(baseArgs, "zone-settings", "get", "ssl", "nonexistent_setting", "--zone", "example.com")...)
		if result.err != nil {
			t.Fatalf("unexpected err=%v stderr=%s", result.err, result.stderr)
		}
		if !strings.Contains(result.stdout, `"@unresolved"`) {
			t.Fatalf("stdout = %s, want @unresolved for missing setting", result.stdout)
		}
	})

	t.Run("waiting-rooms get with --zone flag", func(t *testing.T) {
		result := runCommand(t, append(baseArgs, "waiting-rooms", "get", "wr_mock_sale", "--zone", "example.com")...)
		if result.err != nil || result.stderr != "" {
			t.Fatalf("err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
		}
		if !strings.Contains(result.stdout, `"sale-room"`) {
			t.Fatalf("stdout = %s, want sale-room", result.stdout)
		}
	})

	t.Run("waiting-rooms get missing room emits @unresolved", func(t *testing.T) {
		result := runCommand(t, append(baseArgs, "waiting-rooms", "get", "wr_mock_sale", "no_such_room", "--zone", "example.com")...)
		if result.err != nil {
			t.Fatalf("unexpected err=%v stderr=%s", result.err, result.stderr)
		}
		if !strings.Contains(result.stdout, `"@unresolved"`) {
			t.Fatalf("stdout = %s, want @unresolved for missing room", result.stdout)
		}
		if !strings.Contains(result.stdout, `"sale-room"`) {
			t.Fatalf("stdout = %s, want first room still present", result.stdout)
		}
	})
}

func TestEntityGetCommandLevelError(t *testing.T) {
	baseURL := withMockServer(t)
	// Missing --account-id for account-scoped resource → command-level stderr error, exit 1
	result := runExecute(t, "--base-url", baseURL, "--api-token", "cfut_mock", "workers", "get", "api-worker")
	if result.err == nil {
		t.Fatalf("expected command error when no account is resolvable")
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %s, want empty on command-level error", result.stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.stderr), &payload); err != nil {
		t.Fatalf("stderr is not JSON: %s", result.stderr)
	}
	if payload["fixable_by"] != "agent" {
		t.Fatalf("payload = %#v, want fixable_by:agent", payload)
	}
}

func TestMutationCommandsRequireDryRunOrConfirm(t *testing.T) {
	baseURL := withMockServer(t)
	result := runExecute(t, "--base-url", baseURL, "--api-token", "cfut_mock", "cache", "purge", "example.com", "--url", "https://example.com/a")
	if result.err == nil {
		t.Fatalf("expected command error without dry-run or confirm")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.stderr), &payload); err != nil {
		t.Fatalf("stderr is not JSON: %s", result.stderr)
	}
	if payload["fixable_by"] != "agent" || payload["hint"] == "" {
		t.Fatalf("payload = %#v, want agent error with hint", payload)
	}
}

func TestMutationModeConflictHasPreviewHint(t *testing.T) {
	baseURL := withMockServer(t)
	result := runExecute(t, "--base-url", baseURL, "--api-token", "cfut_mock", "cache", "purge", "example.com", "--url", "https://example.com/a", "--dry-run", "--confirm")
	if result.err == nil {
		t.Fatalf("expected command error with conflicting mutation flags")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.stderr), &payload); err != nil {
		t.Fatalf("stderr is not JSON: %s", result.stderr)
	}
	if !strings.Contains(payload["hint"].(string), "preview") {
		t.Fatalf("payload = %#v, want preview hint", payload)
	}
}

func TestMutationDryRunAndConfirm(t *testing.T) {
	baseURL := withMockServer(t)
	baseArgs := []string{"--base-url", baseURL, "--api-token", "cfut_mock"}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "cache purge dry-run", args: []string{"cache", "purge", "example.com", "--url", "https://example.com/a", "--dry-run"}, want: "dry_run"},
		{name: "cache purge confirm", args: []string{"cache", "purge", "example.com", "--url", "https://example.com/a", "--confirm"}, want: "cache.purge"},
		{name: "dns create dry-run", args: []string{"dns", "create", "example.com", "--type", "CNAME", "--name", "app", "--content", "target.example.com", "--dry-run"}, want: "dry_run"},
		{name: "dns create confirm", args: []string{"dns", "create", "example.com", "--type", "A", "--name", "new.example.com", "--content", "203.0.113.20", "--confirm"}, want: "dns.create"},
		{name: "dns update confirm", args: []string{"dns", "update", "dns_mock_a", "example.com", "--content", "203.0.113.30", "--confirm"}, want: "dns.update"},
		{name: "waiting room update dry-run", args: []string{"waiting-rooms", "update", "wr_mock_sale", "example.com", "--enabled", "--dry-run"}, want: "dry_run"},
		{name: "waiting room update confirm", args: []string{"waiting-rooms", "update", "wr_mock_sale", "example.com", "--enabled", "--confirm"}, want: "waiting_room.update"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append(append([]string{}, baseArgs...), tt.args...)
			result := runCommand(t, args...)
			if result.err != nil || result.stderr != "" {
				t.Fatalf("command err=%v stderr=%s stdout=%s", result.err, result.stderr, result.stdout)
			}
			if !strings.Contains(result.stdout, tt.want) {
				t.Fatalf("stdout = %s, want %s", result.stdout, tt.want)
			}
		})
	}
}
