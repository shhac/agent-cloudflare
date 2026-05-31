package cli

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
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
	tokens map[string]string
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
	delete(m.tokens, name)
}

func runCommand(t *testing.T, args ...string) commandResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	restoreWriters := output.SetWriters(&stdout, &stderr)
	t.Cleanup(restoreWriters)

	cmd := newRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
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
	result := runCommand(t, "--base-url", baseURL, "--api-token", "cfut_mock", "dns", "list")
	if result.err != nil {
		t.Fatalf("Execute() error = %v", result.err)
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
