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
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe() failed: %v", err)
		}
		os.Stdout = w
		defer func() {
			os.Stdout = oldStdout
			w.Close()
			r.Close()
		}()
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

func TestConsole_Getters(t *testing.T) {
	c, cleanup := newTestConsole(t)
	defer cleanup()

	if c.In() == nil {
		t.Error("In() returned nil")
	}
	if c.Out() == nil {
		t.Error("Out() returned nil")
	}
	if c.Err() == nil {
		t.Error("Err() returned nil")
	}
	if !c.IsATTYOut() {
		t.Error("IsATTYOut() returned false, want true")
	}

	if con, ok := c.(*console); ok {
		if !con.IsATTYErr() {
			t.Error("IsATTYErr() returned false, want true")
		}
	} else {
		t.Log("Could not cast to *console to test IsATTYErr")
	}
}

func TestConsole_Size(t *testing.T) {
	c, cleanup := newTestConsole(t)
	defer cleanup()

	consoleImpl, ok := c.(*console)
	if !ok {
		t.Fatal("test console is not of type *console")
	}
	if !term.IsTerminal(int(consoleImpl.out.Fd())) {
		if w, h := c.Size(); w != 0 || h != 0 {
			t.Errorf("Size() on non-TTY should return 0,0, but got %d, %d", w, h)
		}
		t.Skip("Skipping TTY-specific size assertions on non-TTY test console")
	}

	w, h := c.Size()
	if w <= 0 || h <= 0 {
		t.Errorf("Size() returned non-positive dimensions: %d, %d", w, h)
	}
}

func TestConsole_RawMode(t *testing.T) {
	c, cleanup := newTestConsole(t)
	defer cleanup()

	consoleImpl, ok := c.(*console)
	if !ok {
		t.Fatal("test console is not of type *console")
	}
	if !term.IsTerminal(int(consoleImpl.in.Fd())) {
		if _, err := c.MakeRaw(); !errors.Is(err, ErrNotAConsole) {
			t.Errorf("MakeRaw() on non-TTY should return ErrNotAConsole, but got: %v", err)
		}
		t.Skip("Skipping TTY-specific raw mode assertions on non-TTY test console")
	}

	st, err := c.MakeRaw()
	if err != nil {
		t.Fatalf("MakeRaw() failed: %v", err)
	}
	if st == nil {
		t.Fatal("MakeRaw() returned nil state")
	}

	if err := c.Restore(st); err != nil {
		t.Fatalf("Restore() failed: %v", err)
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

func TestConsole_OnResize_NilWatcher(t *testing.T) {
	c := &console{}
	ch := c.OnResize()
	if ch == nil {
		t.Fatal("OnResize() on a console with nil watcher returned nil channel")
	}
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel from OnResize() on a console with nil watcher was not closed")
		}
	default:
		t.Error("channel from OnResize() on a console with nil watcher was not closed")
	}
}
