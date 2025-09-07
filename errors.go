package ptyx

import (
	"errors"
	"fmt"
)

var ErrMuxAlreadyStarted = errors.New("ptyx: mux already started")

type ExitError struct {
	ExitCode int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("process exited with status %d", e.ExitCode)
}
