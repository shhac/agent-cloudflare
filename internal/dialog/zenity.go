package dialog

import (
	"context"
	"errors"
	"fmt"

	"github.com/ncruces/zenity"
)

type zenityPrompter struct{}

func (z *zenityPrompter) Available() error { return platformAvailable() }

func (z *zenityPrompter) Prompt(ctx context.Context, spec Spec) ([]Result, error) {
	if err := validateSpec(spec); err != nil {
		return nil, err
	}
	if len(spec.Items) == 0 {
		return nil, nil
	}
	if err := z.Available(); err != nil {
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
		value, err := promptOne(title, item)
		if err != nil {
			return nil, classifyZenityError(err, item)
		}
		results = append(results, Result{ID: item.ID, Value: value})
	}
	return results, nil
}

func promptOne(title string, item Field) (string, error) {
	switch item.InputType {
	case Password:
		_, value, err := zenity.Password(zenity.Title(title))
		return value, err
	case Text:
		return zenity.Entry(item.Label, zenity.Title(title))
	default:
		return "", fmt.Errorf("dialog: unsupported input type %d for field %q", item.InputType, item.ID)
	}
}

func classifyZenityError(err error, item Field) error {
	if errors.Is(err, zenity.ErrCanceled) {
		return fmt.Errorf("%w (%s)", ErrCancelled, item.Label)
	}
	return fmt.Errorf("dialog failed (%s): %w", item.Label, err)
}
