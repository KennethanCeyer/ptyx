//go:build windows

package ptyx

import (
	"errors"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
	"unsafe"
)

func TestCreateEnvBlock(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want []uint16
	}{
		{
			name: "Simple case",
			env:  []string{"A=B", "C=D"},
			want: utf16.Encode([]rune("A=B\x00C=D\x00\x00")),
		},
		{
			name: "Empty env",
			env:  []string{},
			want: nil,
		},
		{
			name: "Nil env",
			env:  nil,
			want: nil,
		},
		{
			name: "Sanitize NUL character",
			env:  []string{"A=B", "INVALID\x00KEY=val", "C=D"},
			want: utf16.Encode([]rune("A=B\x00C=D\x00\x00")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPtr := createEnvBlock(tt.env)

			if gotPtr == nil {
				if tt.want != nil {
					t.Errorf("createEnvBlock() got nil, want non-nil")
				}
				return
			}
			if tt.want == nil {
				t.Errorf("createEnvBlock() got non-nil, want nil")
				return
			}

			block := (*[1 << 16]uint16)(unsafe.Pointer(gotPtr))[:]
			var got []uint16
			for i := 0; i < len(block)-1; i++ {
				if block[i] == 0 && block[i+1] == 0 {
					got = block[:i+2]
					break
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createEnvBlock() got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWindowsSpawn(t *testing.T) {
	t.Run("NonExistentProgram", func(t *testing.T) {
		_, err := Spawn(SpawnOpts{Prog: "a-program-that-does-not-exist-12345.exe"})
		if err == nil {
			t.Fatal("Spawn with non-existent program should return an error, but got nil")
		}

		var execErr *exec.Error
		if !errors.As(err, &execErr) {
			t.Errorf("Spawn error = %v (type %T), want type *exec.Error", err, err)
		}
	})
}

func TestWinSession_Wait_ExitError(t *testing.T) {
	s, err := Spawn(SpawnOpts{Prog: "cmd.exe", Args: []string{"/c", "exit 42"}})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skipf("could not find 'cmd.exe', skipping test: %v", err)
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

func TestWinSession_Kill(t *testing.T) {
	s, err := Spawn(SpawnOpts{Prog: "ping.exe", Args: []string{"-t", "127.0.0.1"}})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skipf("could not find 'ping.exe', skipping test: %v", err)
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

	if exitErr.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1 for a killed process", exitErr.ExitCode)
	}

	_ = s.Close()
}

func TestWindowsSpawn_WithOptions(t *testing.T) {
	t.Run("Env", func(t *testing.T) {
		s, err := Spawn(SpawnOpts{
			Prog: "cmd.exe",
			Args: []string{"/c", "echo %PTYX_TEST_VAR%"},
			Env:  []string{"PTYX_TEST_VAR=hello_ptyx"},
		})
		if err != nil {
			t.Fatalf("Spawn failed: %v", err)
		}
		defer s.Close()

		output, _ := io.ReadAll(s.PtyReader())
		_ = s.Wait()

		if got := strings.TrimSpace(string(output)); !strings.Contains(got, "hello_ptyx") {
			t.Errorf("Expected output to contain 'hello_ptyx', got %q", got)
		}
	})

	t.Run("Dir", func(t *testing.T) {
		tempDir := t.TempDir()
		s, err := Spawn(SpawnOpts{Prog: "cmd.exe", Args: []string{"/c", "cd"}, Dir: tempDir})
		if err != nil {
			t.Fatalf("Spawn failed: %v", err)
		}
		defer s.Close()

		output, _ := io.ReadAll(s.PtyReader())
		_ = s.Wait()

		if got := strings.TrimSpace(string(output)); !strings.HasSuffix(got, tempDir) {
			t.Errorf("Expected output to end with '%s', got %q", tempDir, got)
		}
	})
}
