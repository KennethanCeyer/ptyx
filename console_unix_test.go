//go:build linux || darwin || freebsd || netbsd || openbsd

package ptyx

import (
	"errors"
	"testing"
)

func newPlatformTestConsole(t *testing.T) (Console, func()) {
	c, err := NewConsole()
	if err != nil {
		if errors.Is(err, ErrNotAConsole) {
			t.Skipf("Not a console, skipping test: %v", err)
		}
		t.Fatalf("NewConsole() failed: %v", err)
	}
	return c, func() { c.Close() }
}
