package auth

import (
	"context"
	"fmt"

	"github.com/shhac/lib-agent-cli/dialog"

	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func promptTokenViaDialog(ctx context.Context, profile, apiToken string) (string, error) {
	if apiToken != "" {
		return apiToken, nil
	}
	spec := dialog.Spec{
		Title: fmt.Sprintf("agent-cloudflare credential: %s", profile),
		Items: []dialog.Field{
			{ID: "api_token", Label: "Cloudflare API token", InputType: dialog.Password},
		},
	}
	if err := dialog.Default.Available(); err != nil {
		return apiToken, classifyDialogErr(err, profile)
	}
	results, err := dialog.Default.Prompt(ctx, spec)
	if err != nil {
		return apiToken, classifyDialogErr(err, profile)
	}
	for _, result := range results {
		if result.ID == "api_token" {
			apiToken = result.Value
		}
	}
	return apiToken, nil
}

func classifyDialogErr(err error, profile string) error {
	cat, hint := dialog.ClassifyError(err)
	switch cat {
	case dialog.CategoryHuman:
		hint = "agent-cloudflare profiles add --form requires a graphical desktop session. " +
			"Ask the user to run it on their local machine, or fall back to non-interactive: " +
			fmt.Sprintf("agent-cloudflare profiles add %s --api-token <secret>", profile)
	case dialog.CategoryRetry:
		hint = "User cancelled the dialog. Re-run agent-cloudflare profiles add --form to retry."
	}
	return agenterrors.Wrap(err, categoryToFixableBy(cat)).WithHint(hint)
}

func categoryToFixableBy(c dialog.Category) agenterrors.FixableBy {
	switch c {
	case dialog.CategoryHuman:
		return agenterrors.FixableByHuman
	case dialog.CategoryRetry:
		return agenterrors.FixableByRetry
	default:
		return agenterrors.FixableByAgent
	}
}
