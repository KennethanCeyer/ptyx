package testptyx

import (
	"errors"
	"io"
	"os"
	"testing"
)

func TestMockConsole(t *testing.T) {
	t.Run("NewMockConsole", func(t *testing.T) {
		input := "hello"
		mc := NewMockConsole(input)
		if mc.OutBuffer == nil {
			t.Error("NewMockConsole: OutBuffer should not be nil")
		}
		if mc.InReader == nil {
			t.Error("NewMockConsole: InReader should not be nil")
		}
		readBytes, _ := io.ReadAll(mc.InReader)
		if string(readBytes) != input {
			t.Errorf("NewMockConsole: expected input %q, got %q", input, string(readBytes))
		}
	})

	t.Run("Out writer", func(t *testing.T) {
		mc := NewMockConsole("")
		testString := "test output"
		_, err := mc.Out().Write([]byte(testString))
		if err != nil {
			t.Fatalf("Write to Out() failed: %v", err)
		}
		if mc.OutBuffer.String() != testString {
			t.Errorf("Out() buffer content = %q, want %q", mc.OutBuffer.String(), testString)
		}
	})

	t.Run("Out with ForceWriteError", func(t *testing.T) {
		mc := NewMockConsole("")
		expectedErr := errors.New("forced write error")
		mc.ForceWriteError = expectedErr
		_, err := mc.Out().Write([]byte("test"))
		if !errors.Is(err, expectedErr) {
			t.Errorf("Out() with ForceWriteError: got err %v, want %v", err, expectedErr)
		}
	})

	t.Run("Err method", func(t *testing.T) {
		mc := NewMockConsole("")
		if mc.Err() != os.Stderr {
			t.Error("Err() should return os.Stderr")
		}
	})

	t.Run("MakeRaw method", func(t *testing.T) {
		mc := NewMockConsole("")
		_, err := mc.MakeRaw()
		if err != nil {
			t.Errorf("MakeRaw() returned unexpected error: %v", err)
		}

		expectedErr := errors.New("forced make raw error")
		mc.MakeRawError = expectedErr
		_, err = mc.MakeRaw()
		if !errors.Is(err, expectedErr) {
			t.Errorf("MakeRaw() with MakeRawError: got err %v, want %v", err, expectedErr)
		}
	})

	t.Run("Simple methods", func(t *testing.T) {
		mc := NewMockConsole("")
		if !mc.IsATTYOut() {
			t.Error("IsATTYOut() should return true")
		}
		if w, h := mc.Size(); w != 80 || h != 24 {
			t.Errorf("Size() = %d, %d, want 80, 24", w, h)
		}
		if err := mc.Restore(nil); err != nil {
			t.Errorf("Restore() returned unexpected error: %v", err)
		}
		if err := mc.Close(); err != nil {
			t.Errorf("Close() returned unexpected error: %v", err)
		}
		mc.EnableVT() // no-op, just for coverage
	})

	t.Run("OnResize", func(t *testing.T) {
		mc := NewMockConsole("")
		ch := mc.OnResize()
		if ch == nil {
			t.Fatal("OnResize() returned nil channel")
		}
		select {
		case _, ok := <-ch:
			if ok {
				t.Error("OnResize() channel should be closed")
			}
		default:
			t.Error("OnResize() channel should be closed immediately")
		}
	})
}

func TestMockSession(t *testing.T) {
	t.Run("NewMockSession", func(t *testing.T) {
		output := "hello"
		ms := NewMockSession(output)
		if ms.PtyInBuffer == nil {
			t.Error("NewMockSession: PtyInBuffer should not be nil")
		}
		if ms.PtyOutReader == nil {
			t.Error("NewMockSession: PtyOutReader should not be nil")
		}
		readBytes, _ := io.ReadAll(ms.PtyOutReader)
		if string(readBytes) != output {
			t.Errorf("NewMockSession: expected output %q, got %q", output, string(readBytes))
		}
	})

	t.Run("PtyWriter", func(t *testing.T) {
		ms := NewMockSession("")
		testString := "test input"
		_, err := ms.PtyWriter().Write([]byte(testString))
		if err != nil {
			t.Fatalf("Write to PtyWriter() failed: %v", err)
		}
		if ms.PtyInBuffer.String() != testString {
			t.Errorf("PtyWriter() buffer content = %q, want %q", ms.PtyInBuffer.String(), testString)
		}
	})

	t.Run("PtyWriter with ForceWriteError", func(t *testing.T) {
		ms := NewMockSession("")
		expectedErr := errors.New("forced pty write error")
		ms.ForceWriteError = expectedErr
		_, err := ms.PtyWriter().Write([]byte("test"))
		if !errors.Is(err, expectedErr) {
			t.Errorf("PtyWriter() with ForceWriteError: got err %v, want %v", err, expectedErr)
		}
	})

	t.Run("Wait", func(t *testing.T) {
		ms := NewMockSession("")
		if err := ms.Wait(); err != nil {
			t.Errorf("Wait() returned unexpected error: %v", err)
		}

		expectedErr := errors.New("forced wait error")
		ms.WaitError = expectedErr
		if err := ms.Wait(); !errors.Is(err, expectedErr) {
			t.Errorf("Wait() with WaitError: got err %v, want %v", err, expectedErr)
		}
	})

	t.Run("Simple methods", func(t *testing.T) {
		ms := NewMockSession("")
		if err := ms.Resize(1, 1); err != nil {
			t.Errorf("Resize() returned unexpected error: %v", err)
		}
		if err := ms.Kill(); err != nil {
			t.Errorf("Kill() returned unexpected error: %v", err)
		}
		if err := ms.Close(); err != nil {
			t.Errorf("Close() returned unexpected error: %v", err)
		}
		if pid := ms.Pid(); pid != 1234 {
			t.Errorf("Pid() = %d, want 1234", pid)
		}
		if err := ms.CloseStdin(); err != nil {
			t.Errorf("CloseStdin() returned unexpected error: %v", err)
		}
	})
}
