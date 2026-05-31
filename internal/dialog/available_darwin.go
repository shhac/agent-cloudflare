//go:build darwin

package dialog

import (
	"fmt"
	"os"
)

func platformAvailable() error {
	if os.Getenv("SSH_CONNECTION") != "" && os.Getenv("TERM_PROGRAM") == "" {
		return fmt.Errorf("%w: appears to be an SSH session with no local terminal", ErrNoGUI)
	}
	return nil
}
