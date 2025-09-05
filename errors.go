package ptyx

import (
	"errors"
	"fmt"
)

var ErrMuxAlreadyStarted = errors.New("ptyx: mux can only be started once")

type ExitError struct {
	ExitCode int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("process exited with status %d", e.ExitCode)
}
