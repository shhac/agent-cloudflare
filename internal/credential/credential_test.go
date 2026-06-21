package credential

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shhac/agent-cloudflare/internal/config"
)

type fakeBackend struct {
	tokens  map[string]string
	deleted []string
	err     error
}

func (f *fakeBackend) Store(name, token string) error {
	if f.err != nil {
		return f.err
	}
	if f.tokens == nil {
		f.tokens = map[string]string{}
	}
	f.tokens[name] = token
	return nil
}

func (f *fakeBackend) Get(name string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.tokens[name], nil
}

func (f *fakeBackend) Delete(name string) {
	f.deleted = append(f.deleted, name)
	delete(f.tokens, name)
}

func withTempCredentialConfig(t *testing.T, backend *fakeBackend) {
	t.Helper()
	config.SetConfigDir(t.TempDir())
	t.Cleanup(func() { config.SetConfigDir("") })
	restore := SetBackendForTest(backend)
	t.Cleanup(restore)
}

func TestStoreGetRemoveKeepsTokenOutOfIndex(t *testing.T) {
	backend := &fakeBackend{}
	withTempCredentialConfig(t, backend)

	storage, err := Store("prod", "cfut_secret")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	if storage != "keychain" {
		t.Fatalf("storage = %q, want keychain", storage)
	}
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		t.Fatalf("ReadFile(credentialsPath) error = %v", err)
	}
	if strings.Contains(string(data), "cfut_secret") {
		t.Fatalf("credentials index leaked token: %s", data)
	}
	if !strings.Contains(string(data), `"keychain_managed": true`) {
		t.Fatalf("credentials index = %s, want keychain metadata", data)
	}

	token, err := Get("prod")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if token != "cfut_secret" {
		t.Fatalf("token = %q", token)
	}

	if err := Remove("prod"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if len(backend.deleted) != 1 || backend.deleted[0] != "prod" {
		t.Fatalf("deleted = %#v", backend.deleted)
	}
	if _, err := Get("prod"); err == nil {
		t.Fatalf("Get() after Remove error = nil")
	}
}

func TestMissingCredentialReturnsNotFound(t *testing.T) {
	withTempCredentialConfig(t, &fakeBackend{})

	_, err := Get("missing")
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Get() error = %T %[1]v, want NotFoundError", err)
	}
}

func TestStoreBackendFailureFallsBackToFile(t *testing.T) {
	withTempCredentialConfig(t, &fakeBackend{err: errors.New("keychain unavailable")})

	storage, err := Store("prod", "cfut_secret")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	if storage != "file" {
		t.Fatalf("storage = %q, want file", storage)
	}
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		t.Fatalf("ReadFile(credentialsPath) error = %v", err)
	}
	if !strings.Contains(string(data), "cfut_secret") {
		t.Fatalf("credentials index should hold the raw token under keychain failure: %s", data)
	}
	if strings.Contains(string(data), keychainSentinel) {
		t.Fatalf("credentials index should not hold the sentinel under keychain failure: %s", data)
	}

	token, err := Get("prod")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if token != "cfut_secret" {
		t.Fatalf("token = %q, want cfut_secret", token)
	}
}

// TestStore_Headless_FileFallback exercises the credential-WRITE path
// non-interactively. The per-CLI keychain opt-out (derived by lib-agent-cli from
// the "app.paulie.agent-cloudflare" service) makes the keychain report
// unavailable, so Store deterministically keeps the raw token in the 0600 index
// file on every platform — including darwin, where it would otherwise reach the
// `security` GUI prompt. Before the file fallback existed, Store simply failed
// under the opt-out (and on any non-macOS host).
func TestStore_Headless_FileFallback(t *testing.T) {
	t.Setenv("AGENT_CLOUDFLARE_NO_KEYCHAIN", "1")
	dir := t.TempDir()
	config.SetConfigDir(dir)
	t.Cleanup(func() { config.SetConfigDir("") })

	storage, err := Store("headless", "cfut-headless-token")
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if storage != "file" {
		t.Fatalf("storage=%q, want \"file\" (keychain opt-out should force the file path)", storage)
	}

	path := filepath.Join(dir, "credentials.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("index not written: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("index mode=%o, want 0600", mode)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "cfut-headless-token") {
		t.Errorf("file should contain the raw token under opt-out; got %s", data)
	}
	if strings.Contains(string(data), keychainSentinel) {
		t.Errorf("file should NOT contain the keychain sentinel under opt-out; got %s", data)
	}

	got, err := Get("headless")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "cfut-headless-token" {
		t.Errorf("Get=%q, want cfut-headless-token", got)
	}

	if err := Remove("headless"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := Get("headless"); err == nil {
		t.Error("expected NotFound after Remove")
	}
}
