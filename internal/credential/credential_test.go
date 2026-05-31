package credential

import (
	"errors"
	"os"
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

func TestStoreBackendFailureDoesNotWriteIndex(t *testing.T) {
	withTempCredentialConfig(t, &fakeBackend{err: errors.New("keychain unavailable")})

	if _, err := Store("prod", "cfut_secret"); err == nil {
		t.Fatalf("Store() error = nil")
	}
	if data, err := os.ReadFile(credentialsPath()); err == nil {
		t.Fatalf("credentials index should not be written after backend failure: %s", data)
	} else if !os.IsNotExist(err) {
		t.Fatalf("ReadFile(credentialsPath) error = %v, want not exist", err)
	}
}
