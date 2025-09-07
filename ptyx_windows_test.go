//go:build windows

package ptyx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

func TestWinHelperProcess(t *testing.T) {
	if os.Getenv("PTYX_HELPER") != "1" {
		return
	}
	switch os.Getenv("MODE") {
	case "env":
		fmt.Println(os.Getenv("PTYX_TEST_VAR"))
	case "dir":
		wd, _ := os.Getwd()
		fmt.Println(wd)
	case "exit":
		os.Exit(17)
	default:
		fmt.Println("noop")
	}
	_ = os.Stdout.Sync()
}

func TestBuildEnvBlock(t *testing.T) {
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
			got := buildEnvBlock(tt.env)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildEnvBlock() got %v, want %v", got, tt.want)
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
	s, err := Spawn(SpawnOpts{
		Prog: os.Args[0],
		Args: []string{"-test.run=^TestWinHelperProcess$"},
		Env: append(os.Environ(),
			"PTYX_HELPER=1",
			"MODE=exit",
		),
	})
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	defer s.Close()

	waitErr := s.Wait()
	if waitErr == nil {
		t.Fatal("Wait() returned nil, expected an ExitError")
	}
	var exitErr *ExitError
	if !errors.As(waitErr, &exitErr) || exitErr.ExitCode != 17 {
		t.Fatalf("Wait() error = %v (type %T), want *ptyx.ExitError with code 17", waitErr, waitErr)
	}
}

func TestWinSession_Kill(t *testing.T) {
	s, err := Spawn(SpawnOpts{
		Prog: "powershell",
		Args: []string{"-Command", "while ($true) { Write-Host 'looping...'; Start-Sleep -Milliseconds 100 }"},
	})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skipf("could not find 'powershell.exe', skipping test: %v", err)
		}
		t.Fatalf("Spawn failed: %v", err)
	}
	defer s.Close()

	go io.Copy(io.Discard, s.PtyReader())

	if err := s.Kill(); err != nil {
		t.Fatalf("Kill() failed: %v", err)
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- s.Wait() }()

	select {
	case waitErr := <-waitDone:
		if waitErr == nil {
			t.Fatal("Wait() returned nil, expected an error after Kill()")
		}
		var exitErr *ExitError
		if !errors.As(waitErr, &exitErr) || exitErr.ExitCode != 1 {
			t.Fatalf("Wait() error = %v (type %T), want *ptyx.ExitError with code 1", waitErr, waitErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("s.Wait() timed out after s.Kill()")
	}
}

func TestWindowsSpawn_WithOptions(t *testing.T) {
	t.Run("Env", func(t *testing.T) {
		line, err := spawnReadOneLineAndCloseWin(SpawnOpts{
			Prog: os.Args[0],
			Args: []string{"-test.run=^TestWinHelperProcess$"},
			Env: append(os.Environ(),
				"PTYX_HELPER=1",
				"MODE=env",
				"PTYX_TEST_VAR=hello_ptyx",
			),
		}, 5*time.Second)
		if err != nil {
			t.Fatalf("Spawn failed: %v", err)
		}
		if !strings.Contains(line, "hello_ptyx") {
			t.Errorf("Expected output to contain 'hello_ptyx', got %q", line)
		}
	})

	t.Run("Dir", func(t *testing.T) {
		tempDir := t.TempDir()
		line, err := spawnReadOneLineAndCloseWin(SpawnOpts{
			Prog: os.Args[0],
			Args: []string{"-test.run=^TestWinHelperProcess$"},
			Env:  append(os.Environ(), "PTYX_HELPER=1", "MODE=dir"),
			Dir:  tempDir,
		}, 5*time.Second)
		if err != nil {
			t.Fatalf("Spawn failed: %v", err)
		}
		if !strings.HasSuffix(strings.TrimSpace(line), tempDir) {
			t.Errorf("Expected output to end with %q, got %q", tempDir, line)
		}
	})
}

var (
	reCSI = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	reOSC = regexp.MustCompile(`\x1b\].*?(?:\x07|\x1b\\)`)
)

func stripANSI(s string) string {
	s = reCSI.ReplaceAllString(s, "")
	s = reOSC.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func spawnReadOneLineAndCloseWin(opts SpawnOpts, timeout time.Duration) (string, error) {
	s, err := Spawn(opts)
	if err != nil {
		return "", err
	}
	defer s.Close()

	ws, ok := s.(*winSession)
	if !ok {
		return "", errors.New("spawned session is not a *winSession")
	}

	var outputBuf bytes.Buffer
	readerDone := make(chan struct{})
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := s.PtyReader().Read(buf)
			if n > 0 { outputBuf.Write(buf[:n]) }
			if err != nil { break }
		}
		close(readerDone)
	}()

	waitResult, err := windows.WaitForSingleObject(ws.process, uint32(timeout/time.Millisecond))
	if err != nil {
		_ = s.Kill()
		return "", fmt.Errorf("WaitForSingleObject failed: %w", err)
	}

	if waitResult == uint32(windows.WAIT_TIMEOUT) {
		_ = s.Kill()
		return "", fmt.Errorf("process wait timed out after %v", timeout)
	}

	_ = s.Close()

	select {
	case <-readerDone:
	case <-time.After(2 * time.Second):
		return "", errors.New("timeout waiting for PTY reader to finish after process exit")
	}

	output := stripANSI(outputBuf.String())
	parts := strings.Split(output, "\n")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0]), nil
	}
	return "", nil
}
