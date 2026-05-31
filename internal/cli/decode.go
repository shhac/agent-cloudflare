package cli

import (
	"encoding/json"

	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func decodeRaw(raw json.RawMessage) (any, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent).
			WithHint("Cloudflare returned JSON the CLI could not decode for this command; retry with --debug")
	}
	return decoded, nil
}
