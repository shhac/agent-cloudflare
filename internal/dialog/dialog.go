package dialog

import (
	"context"
	"errors"
	"fmt"
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

var Default Prompter = &zenityPrompter{}

func SetDefault(p Prompter) (restore func()) {
	prev := Default
	Default = p
	return func() { Default = prev }
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
