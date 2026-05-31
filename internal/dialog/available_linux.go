//go:build linux

package dialog

import (
	"fmt"
	"os"
)

func platformAvailable() error {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("%w: DISPLAY/WAYLAND_DISPLAY is not set", ErrNoGUI)
	}
	return nil
}
