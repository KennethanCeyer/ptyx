//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package ptyx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("PTYX_HELPER") != "1" {
		return
	}
	switch os.Getenv("MODE") {
	case "env":
		fmt.Println(os.Getenv("PTYX_TEST_VAR"))
	case "dir":
		wd, _ := os.Getwd()
		fmt.Println(wd)
	default:
		fmt.Println("noop")
	}
	_ = os.Stdout.Sync()
}

func TestUnixSpawn(t *testing.T) {
	t.Run("EmptyProgram", func(t *testing.T) {
		_, err := Spawn(context.Background(), SpawnOpts{Prog: ""})
		if err == nil {
			t.Fatal("Spawn with empty program should return an error, but got nil")
		}
		if err.Error() != "ptyx: empty program" {
			t.Errorf("Spawn error = %q, want %q", err.Error(), "ptyx: empty program")
		}
	})

	t.Run("NonExistentProgram", func(t *testing.T) {
		_, err := Spawn(context.Background(), SpawnOpts{Prog: "a-program-that-does-not-exist-12345"})
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
		line, err := spawnReadOneLineAndClose(context.Background(), SpawnOpts{
			Prog: os.Args[0],
			Args: []string{"-test.run=^TestHelperProcess$"},
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
		line, err := spawnReadOneLineAndClose(context.Background(), SpawnOpts{
			Prog: os.Args[0],
			Args: []string{"-test.run=^TestHelperProcess$"},
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := clen(tt.b); got != tt.want {
				t.Errorf("clen() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnixSession_Wait_ExitError(t *testing.T) {
	s, err := Spawn(context.Background(), SpawnOpts{Prog: "sh", Args: []string{"-c", "exit 17"}})
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
	if !errors.As(waitErr, &exitErr) || exitErr.ExitCode != 17 {
		t.Fatalf("Wait() error = %v (type %T), want *ptyx.ExitError with code 17", waitErr, waitErr)
	}
}

func TestUnixSession_Kill(t *testing.T) {
	s, err := Spawn(context.Background(), SpawnOpts{Prog: "sh", Args: []string{"-c", `while true; do echo "looping..."; sleep 0.1; done`}})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skipf("could not find 'sh', skipping test: %v", err)
		}
		t.Fatalf("Spawn failed: %v", err)
	}
	defer s.Close()

	doneCopy := make(chan struct{})
	go func() {
		io.Copy(io.Discard, s.PtyReader())
		close(doneCopy)
	}()

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
		if !errors.As(waitErr, &exitErr) || exitErr.ExitCode != -1 {
			t.Fatalf("Wait() error = %v (type %T), want *ptyx.ExitError with code -1", waitErr, waitErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("s.Wait() timed out after s.Kill()")
	}

	select {
	case <-doneCopy:
	case <-time.After(2 * time.Second):
		t.Fatal("reader did not finish in time")
	}
}

func spawnReadOneLineAndClose(ctx context.Context, opts SpawnOpts, timeout time.Duration) (string, error) {
	s, err := Spawn(ctx, opts)
	if err != nil {
		return "", err
	}

	type lineRes struct {
		line string
		err  error
	}
	lineCh := make(chan lineRes, 1)
	waitCh := make(chan error, 1)

	go func() {
		line, rerr := readPTYOneLine(s.PtyReader())
		lineCh <- lineRes{line: strings.TrimRight(line, "\r\n"), err: rerr}
	}()
	go func() { waitCh <- s.Wait() }()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var line string
	select {
	case lr := <-lineCh:
		if lr.err != nil && !isPTYEOF(lr.err) && !errors.Is(lr.err, io.EOF) {
			_ = s.Kill()
			_ = s.Close()
			return "", lr.err
		}
		line = lr.line
	case werr := <-waitCh:
		_ = s.Close()
		lr := <-lineCh
		if lr.err != nil && !isPTYEOF(lr.err) && !errors.Is(lr.err, io.EOF) {
			return "", lr.err
		}
		line = lr.line
		if line == "" {
			return "", fmt.Errorf("child exited before producing a line (wait err: %v)", werr)
		}
		return line, nil
	case <-timer.C:
		_ = s.Kill()
		_ = s.Close()
		return "", fmt.Errorf("timeout reading line")
	}

	_ = s.Close()

	select {
	case <-waitCh:
	case <-time.After(timeout):
		_ = s.Kill()
		return "", fmt.Errorf("timeout waiting child")
	}

	return line, nil
}

func readPTYOneLine(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	var buf bytes.Buffer
	for {
		b, err := br.ReadByte()
		if err != nil {
			if isPTYEOF(err) || errors.Is(err, io.EOF) {
				return buf.String(), err
			}
			return "", err
		}
		buf.WriteByte(b)
		if b == '\n' {
			return buf.String(), nil
		}
	}
}

func isPTYEOF(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && (errno == syscall.EIO || errno == 0)
}
