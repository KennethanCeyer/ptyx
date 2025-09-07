//go:build unix

package main

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/KennethanCeyer/ptyx"
)

func checkSignal(err error) string {
	var exitErr *ptyx.ExitError
	if errors.As(err, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return fmt.Sprintf("Terminated by signal: %s\n", ws.Signal())
		}
	}
	return ""
}
