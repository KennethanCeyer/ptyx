//go:build linux || darwin || freebsd || netbsd || openbsd

package ptyx

import (
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

func TestUnixSpawn(t *testing.T) {
	t.Run("EmptyProgram", func(t *testing.T) {
		_, err := Spawn(SpawnOpts{Prog: ""})
		if err == nil {
			t.Fatal("Spawn with empty program should return an error, but got nil")
		}
		if err.Error() != "ptyx: empty program" {
			t.Errorf("Spawn error = %q, want %q", err.Error(), "ptyx: empty program")
		}
	})

	t.Run("NonExistentProgram", func(t *testing.T) {
		_, err := Spawn(SpawnOpts{Prog: "a-program-that-does-not-exist-12345"})
		if err == nil {
			t.Fatal("Spawn with non-existent program should return an error, but got nil")
		}

		var execErr *exec.Error
		if !errors.As(err, &execErr) {
			t.Errorf("Spawn error = %v (type %T), want type *exec.Error", err, err)
		}
	})
}

func TestUnixSpawn_WithOptions(t *testing.T) {
	t.Run("Env", func(t *testing.T) {
		s, err := Spawn(SpawnOpts{
			Prog: "sh",
			Args: []string{"-c", "echo $PTYX_TEST_VAR"},
			Env:  []string{"PTYX_TEST_VAR=hello_ptyx"},
		})
		if err != nil {
			t.Fatalf("Spawn failed: %v", err)
		}
		defer s.Close()

		output, _ := io.ReadAll(s.PtyReader())
		_ = s.Wait()

		if got := strings.TrimSpace(string(output)); got != "hello_ptyx" {
			t.Errorf("Expected output to be 'hello_ptyx', got %q", got)
		}
	})

	t.Run("Dir", func(t *testing.T) {
		tempDir := t.TempDir()
		s, err := Spawn(SpawnOpts{Prog: "pwd", Dir: tempDir})
		if err != nil {
			t.Fatalf("Spawn failed: %v", err)
		}
		defer s.Close()

		output, _ := io.ReadAll(s.PtyReader())
		_ = s.Wait()

		if got := strings.TrimSpace(string(output)); got != tempDir {
			t.Errorf("Expected output to be '%s', got %q", tempDir, got)
		}
	})
}

func TestClen(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		want int
	}{
		{"NUL in middle", []byte{'a', 'b', 0, 'd'}, 2},
		{"NUL at start", []byte{0, 'a', 'b'}, 0},
		{"NUL at end", []byte{'a', 'b', 0}, 2},
		{"No NUL", []byte{'a', 'b', 'c'}, 3},
		{"All NULs", []byte{0, 0, 0}, 0},
		{"Empty slice", []byte{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clen(tt.b); got != tt.want {
				t.Errorf("clen() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnixSession_Wait_ExitError(t *testing.T) {
	s, err := Spawn(SpawnOpts{Prog: "sh", Args: []string{"-c", "exit 42"}})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skipf("could not find 'sh', skipping test: %v", err)
		}
		t.Fatalf("Spawn failed: %v", err)
	}
	defer s.Close()

	waitErr := s.Wait()
	if waitErr == nil {
		t.Fatal("Wait() returned nil, expected an ExitError")
	}
	var exitErr *ExitError
	if !errors.As(waitErr, &exitErr) || exitErr.ExitCode != 42 {
		t.Fatalf("Wait() error = %v (type %T), want *ptyx.ExitError with code 42", waitErr, waitErr)
	}
}

func TestUnixSession_Kill(t *testing.T) {
	s, err := Spawn(SpawnOpts{Prog: "sleep", Args: []string{"30"}})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skipf("could not find 'sleep', skipping test: %v", err)
		}
		t.Fatalf("Spawn failed: %v", err)
	}

	if err := s.Kill(); err != nil {
		t.Fatalf("Kill() failed: %v", err)
	}

	waitErr := s.Wait()
	if waitErr == nil {
		t.Fatal("Wait() returned nil, expected an error after Kill()")
	}

	var exitErr *ExitError
	if !errors.As(waitErr, &exitErr) {
		t.Fatalf("Wait() error = %v (type %T), want *ptyx.ExitError", waitErr, waitErr)
	}

	if exitErr.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1 for a killed process", exitErr.ExitCode)
	}

	_ = s.Close()
}
