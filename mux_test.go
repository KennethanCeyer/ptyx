package ptyx

import (
	"errors"
	"testing"
)

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("i am a bad reader")
}

func TestMux(t *testing.T) {
	t.Run("ConsoleToPty", func(t *testing.T) {
		consoleInput := "hello from console"
		c := newMockConsole(consoleInput)
		s := newMockSession("")

		m := NewMux()
		if err := m.Start(c, s); err != nil {
			t.Fatalf("Mux.Start() failed: %v", err)
		}

		if err := m.Stop(); err != nil {
			t.Fatalf("Mux.Stop() failed: %v", err)
		}

		if got := s.ptyIn.String(); got != consoleInput {
			t.Errorf("pty input = %q, want %q", got, consoleInput)
		}
	})

	t.Run("PtyToConsole", func(t *testing.T) {
		ptyOutput := "hello from pty"
		c := newMockConsole("")
		s := newMockSession(ptyOutput)

		m := NewMux()
		if err := m.Start(c, s); err != nil {
			t.Fatalf("Mux.Start() failed: %v", err)
		}

		if err := m.Stop(); err != nil {
			t.Fatalf("Mux.Stop() failed: %v", err)
		}

		if got := c.outBuf.String(); got != ptyOutput {
			t.Errorf("console output = %q, want %q", got, ptyOutput)
		}
	})

	t.Run("Stop without Start", func(t *testing.T) {
		m := NewMux()
		if err := m.Stop(); err != nil {
			t.Errorf("Stop() on a non-started Mux returned error: %v", err)
		}
	})

	t.Run("StartTwice", func(t *testing.T) {
		c := newMockConsole("")
		s := newMockSession("")
		m := NewMux()

		err1 := m.Start(c, s)
		if err1 != nil {
			t.Fatalf("first Start() failed: %v", err1)
		}

		err2 := m.Start(c, s)
		if err2 == nil {
			t.Fatal("second Start() should have failed, but got nil error")
		}
		if !errors.Is(err2, ErrMuxAlreadyStarted) {
			t.Errorf("second Start() error = %v, want %v", err2, ErrMuxAlreadyStarted)
		}

		if err := m.Stop(); err != nil {
			t.Fatalf("Stop() failed: %v", err)
		}
	})

	t.Run("StopTwice", func(t *testing.T) {
		c := newMockConsole("input")
		s := newMockSession("output")
		m := NewMux()

		if err := m.Start(c, s); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}

		if err := m.Stop(); err != nil {
			t.Errorf("first Stop() failed: %v", err)
		}
		if err := m.Stop(); err != nil {
			t.Errorf("second Stop() failed: %v", err)
		}
	})

	t.Run("FailingPtyReader", func(t *testing.T) {
		c := newMockConsole("")
		s := newMockSession("")
		s.ptyOut = &errorReader{}

		m := NewMux()
		if err := m.Start(c, s); err != nil {
			t.Fatalf("Mux.Start() failed: %v", err)
		}

		if err := m.Stop(); err != nil {
			t.Fatalf("Mux.Stop() failed: %v", err)
		}
	})
}
