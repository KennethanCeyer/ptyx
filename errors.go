package ptyx

import (
	"errors"
	"fmt"
)

var ErrMuxAlreadyStarted = errors.New("ptyx: mux already started")

type ExitError struct {
	ExitCode int
	waitStatus any
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("process exited with status %d", e.ExitCode)
}

func (e *ExitError) Sys() any {
	return e.waitStatus
}
