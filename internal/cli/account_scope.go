package cli

import (
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func requireAccountID(resolved *shared.ResolvedProfile) (string, error) {
	if resolved.AccountID != "" {
		return resolved.AccountID, nil
	}
	return "", agenterrors.New("--account-id is required", agenterrors.FixableByAgent).
		WithHint("Pass --account-id or store one with 'agent-cloudflare profiles update <profile> --account-id <account_id>'")
}
