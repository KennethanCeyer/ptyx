//go:build linux || darwin || freebsd || netbsd || openbsd

package ptyx

import (
	"os"
	"testing"
)

func newPlatformTestConsole(t *testing.T) (Console, func()) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	c := &console{in: r, out: w, err: w, outTTY: true}
	c.initResizeWatcher()
	return c, func() { c.Close(); r.Close(); w.Close() }
}
