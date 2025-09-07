//go:build linux || darwin || freebsd || netbsd || openbsd

package ptyx

import (
	"os"
	"syscall"
	"testing"
	"time"
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

func TestUnixConsole_ResizeSignal(t *testing.T) {
	c, cleanup := newTestConsole(t)
	defer cleanup()

	if c, ok := c.(*console); ok {
		select {
		case <-c.win.ready:
		case <-time.After(1 * time.Second):
			t.Fatal("resize watcher failed to start in time")
		}
	}
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}
	if err := proc.Signal(syscall.SIGWINCH); err != nil {
		t.Skipf("Failed to send SIGWINCH, skipping test: %v", err)
	}

	select {
	case <-c.OnResize():
	case <-time.After(2 * time.Second):
		t.Fatal("OnResize() did not receive a signal within 2s")
	}
}
