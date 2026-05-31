//go:build !darwin && !linux && !windows

package dialog

import "fmt"

func platformAvailable() error {
	return fmt.Errorf("%w: no dialog backend for this platform", ErrUnsupported)
}
