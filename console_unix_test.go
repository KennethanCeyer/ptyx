//go:build linux || darwin || freebsd || netbsd || openbsd

package ptyx

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func newPlatformTestConsole(t *testing.T) (Console, func()) {
	master, slave, err := openPTY()
	if err != nil {
		t.Fatalf("failed to open pty: %v", err)
	}
	if err := setWinsize(int(master.Fd()), 80, 24); err != nil {
		t.Logf("failed to set pty size, continuing anyway: %v", err)
	}

	c := &console{in: slave, out: slave, err: slave, outTTY: true, errTTY: true}
	c.initResizeWatcher()
	return c, func() {
		c.Close()
		master.Close()
		slave.Close()
	}
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
