package cli

import (
	"context"
	"encoding/json"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

type mutationRequest struct {
	DryRun  bool
	Confirm bool
	Method  string
	Path    string
	Body    map[string]any
	Action  string
	Send    func(context.Context) (json.RawMessage, error)
}

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

func executeMutation(ctx context.Context, client *api.Client, flags *shared.GlobalFlags, req mutationRequest) error {
	if err := requireMutationMode(req.DryRun, req.Confirm); err != nil {
		return err
	}
	if req.DryRun {
		writeDryRun(client, flags, req.Method, req.Path, req.Body)
		return nil
	}
	raw, err := req.Send(ctx)
	if err != nil {
		return err
	}
	decoded, err := decodeRaw(raw)
	if err != nil {
		return err
	}
	shared.WriteItem(mutationResult(req.Action, decoded), flags.Format)
	return nil
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

func mustJSON(value any) json.RawMessage {
	b, _ := json.Marshal(value)
	return b
}
