package auth

import "github.com/shhac/agent-cloudflare/internal/credential"

var (
	credentialStore  = credential.Store
	credentialRemove = credential.Remove
)
