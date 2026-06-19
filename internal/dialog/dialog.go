// Package dialog delegates the native secret-entry boilerplate (the zenity
// entry backend) to lib-agent-cli/dialog, while keeping agent-cloudflare's local
// classification surface: a Prompter the auth flow can swap in tests, sentinel
// errors, and a ClassifyError that distinguishes a user cancellation (retry)
// from a headless host (human). The lib reports both as fixable_by:human and
// gives no way to tell them apart, so this package keeps the platform-
// availability check and cancel detection local to preserve that distinction.
// (Migration shim.)
package dialog

import (
	"context"
	"errors"
	"fmt"

	"github.com/ncruces/zenity"
	clidialog "github.com/shhac/lib-agent-cli/dialog"
)

type InputType int

const (
	Text InputType = iota
	Password
)

type Field struct {
	ID        string
	Label     string
	InputType InputType
	Initial   string
}

type Spec struct {
	Title string
	Items []Field
}

type Result struct {
	ID    string
	Value string
}

var (
	ErrCancelled   = errors.New("cancelled by user")
	ErrNoGUI       = errors.New("no GUI dialog available")
	ErrUnsupported = errors.New("platform unsupported")
)

type Category string

const (
	CategoryHuman Category = "human"
	CategoryRetry Category = "retry"
	CategoryAgent Category = "agent"
)

type Prompter interface {
	Prompt(ctx context.Context, spec Spec) ([]Result, error)
	Available() error
}

var Default Prompter = &libPrompter{}

func SetDefault(p Prompter) (restore func()) {
	prev := Default
	Default = p
	return func() { Default = prev }
}

// libPrompter routes each field through lib-agent-cli/dialog, preserving the
// per-field prompting and multi-step titling the local backend did. Availability
// and cancel classification stay local so ClassifyError keeps its human/retry
// split.
type libPrompter struct{}

func (libPrompter) Available() error { return platformAvailable() }

func (p libPrompter) Prompt(ctx context.Context, spec Spec) ([]Result, error) {
	if err := validateSpec(spec); err != nil {
		return nil, err
	}
	if len(spec.Items) == 0 {
		return nil, nil
	}
	if err := p.Available(); err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(spec.Items))
	for i, item := range spec.Items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		title := spec.Title
		if len(spec.Items) > 1 {
			title = fmt.Sprintf("%s (step %d of %d)", spec.Title, i+1, len(spec.Items))
		}
		values, err := clidialog.Prompt(ctx, clidialog.Spec{
			Title:  title,
			Fields: []clidialog.Field{toLibField(item)},
		})
		if err != nil {
			return nil, classifyPromptError(err, item)
		}
		results = append(results, Result{ID: item.ID, Value: values[item.ID]})
	}
	return results, nil
}

func toLibField(item Field) clidialog.Field {
	return clidialog.Field{
		ID:      item.ID,
		Label:   item.Label,
		Hidden:  item.InputType == Password,
		Initial: item.Initial,
	}
}

// classifyPromptError normalizes a lib prompt failure to the local sentinels.
// The lib wraps zenity's cancel error, so errors.Is still sees it through the
// fixable_by:human wrapper.
func classifyPromptError(err error, item Field) error {
	if errors.Is(err, zenity.ErrCanceled) {
		return fmt.Errorf("%w (%s)", ErrCancelled, item.Label)
	}
	return fmt.Errorf("dialog failed (%s): %w", item.Label, err)
}

func ClassifyError(err error) (Category, string) {
	switch {
	case err == nil:
		return CategoryAgent, ""
	case errors.Is(err, ErrCancelled):
		return CategoryRetry, "User cancelled the dialog. Re-run to retry."
	case errors.Is(err, ErrNoGUI), errors.Is(err, ErrUnsupported):
		return CategoryHuman, "A graphical desktop session is required. Run on the user's local machine, or fall back to a non-interactive flow."
	default:
		return CategoryAgent, ""
	}
}

func validateSpec(spec Spec) error {
	for _, item := range spec.Items {
		if item.InputType != Text && item.InputType != Password {
			return fmt.Errorf("dialog: invalid InputType %d for field %q", item.InputType, item.ID)
		}
	}
	return nil
}
