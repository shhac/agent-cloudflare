package cli

import (
	"encoding/json"

	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func decodeRaw(raw json.RawMessage) (any, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return decoded, nil
}
