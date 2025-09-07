package ptyx

import (
	"errors"
	"os"
	"testing"
	"time"

	"golang.org/x/term"
)

func newTestConsole(t *testing.T) (Console, func()) {
	return newPlatformTestConsole(t)
}

func TestNewConsole_NotAConsole(t *testing.T) {
	if term.IsTerminal(int(os.Stdout.Fd())) {
	}

	_, err := NewConsole()
	if !errors.Is(err, ErrNotAConsole) {
		t.Errorf("NewConsole() in non-TTY environment should return ErrNotAConsole, got %v", err)
	}
}

func TestConsole_Close(t *testing.T) {
	c, cleanup := newTestConsole(t)
	defer cleanup()

	select {
	case _, ok := <-c.OnResize():
		if !ok {
			t.Fatal("OnResize channel was closed prematurely")
		}
	case <-time.After(1 * time.Second):
	}

	if err := c.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	select {
	case _, ok := <-c.OnResize():
		if ok {
			t.Error("OnResize() channel was not closed after Close()")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("OnResize() channel did not close within 1s after Close()")
	}

	if err := c.Close(); err != nil {
		t.Errorf("second Close() failed: %v", err)
	}
}

func TestConsole_Restore_NilState(t *testing.T) {
	c, cleanup := newTestConsole(t)
	defer cleanup()

	if err := c.Restore(nil); err != nil {
		t.Errorf("Restore(nil) returned an error: %v", err)
	}
}

func TestConsole_MakeRaw_NotAConsole(t *testing.T) {
	r, _, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	defer r.Close()

	c := &console{in: r}

	_, err = c.MakeRaw()
	if !errors.Is(err, ErrNotAConsole) {
		t.Errorf("MakeRaw on non-TTY should return ErrNotAConsole, got %v", err)
	}
}
