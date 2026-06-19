package credential

import "github.com/shhac/lib-agent-cli/creds"

const keychainService = "app.paulie.agent-cloudflare"

// keychainBackend adapts creds.Keychain to the local backend interface,
// preserving the not-found-as-error contract callers rely on.
type keychainBackend struct {
	kc *creds.Keychain
}

func newKeychainBackend() keychainBackend {
	return keychainBackend{kc: creds.NewKeychain(keychainService)}
}

func (b keychainBackend) Store(name, token string) error {
	return b.kc.Set(name, token)
}

func (b keychainBackend) Get(name string) (string, error) {
	value, ok := b.kc.Get(name)
	if !ok {
		return "", &NotFoundError{Name: name}
	}
	return value, nil
}

func (b keychainBackend) Delete(name string) {
	_ = b.kc.Delete(name)
}
