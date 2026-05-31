package cli

import (
	"encoding/json"
	"net/http"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func requireMutationMode(dryRun, confirm bool) error {
	if dryRun && confirm {
		return agenterrors.New("--dry-run and --confirm cannot be used together", agenterrors.FixableByAgent).
			WithHint("Use --dry-run to preview the request, then rerun with --confirm only after user approval")
	}
	if dryRun || confirm {
		return nil
	}
	return agenterrors.New("mutation requires --dry-run or --confirm", agenterrors.FixableByAgent).
		WithHint("Use --dry-run to preview the request, or --confirm to send it after explicit user approval")
}

func writeDryRun(client *api.Client, flags *shared.GlobalFlags, method, path string, body map[string]any) {
	shared.WriteItem(map[string]any{
		"status":  "dry_run",
		"request": client.PreviewRequest(method, path, mustJSON(body)),
	}, flags.Format)
}

func mutationResult(action string, result any) map[string]any {
	return map[string]any{
		"status": "sent",
		"action": action,
		"result": result,
	}
}

var _ = http.MethodPost

func mustJSON(value any) json.RawMessage {
	b, _ := json.Marshal(value)
	return b
}
