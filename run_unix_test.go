//go:build unix

package ptyx

import (
	"errors"
	"syscall"
)

func isExpectedWaitErrorAfterPTYClose(err error) bool {
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return true
		}
	}
	return false
}
