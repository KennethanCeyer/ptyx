//go:build windows

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
	c := &console{in: r, out: w, err: w}
	c.outTTY = true
	c.initWinWatcher()
	return c, func() { r.Close(); w.Close() }
}
