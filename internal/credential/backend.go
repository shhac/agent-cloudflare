package credential

type backend interface {
	Store(name, token string) error
	Get(name string) (string, error)
	Delete(name string)
}

var keychain backend = securityKeychain{}

func SetBackendForTest(b backend) func() {
	previous := keychain
	keychain = b
	return func() { keychain = previous }
}
