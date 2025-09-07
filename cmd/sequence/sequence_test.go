package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

type eofSignalingReader struct {
	io.Reader
	once sync.Once
	done chan struct{}
}

func (r *eofSignalingReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err != nil {
		r.once.Do(func() { close(r.done) })
	}
	return
}

type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

type mockSequenceSession struct {
	ptyIn   *bytes.Buffer
	ptyOut  io.Reader
	waitErr error
	eofChan chan struct{}
	forceWriteError error
}

func (m *mockSequenceSession) PtyReader() io.Reader      { return m.ptyOut }
func (m *mockSequenceSession) PtyWriter() io.Writer {
	if m.forceWriteError != nil {
		return &errorWriter{err: m.forceWriteError}
	}
	return m.ptyIn
}
func (m *mockSequenceSession) Resize(cols, rows int) error { return nil }
func (m *mockSequenceSession) Wait() error {
	if m.eofChan != nil {
		<-m.eofChan
	}
	return m.waitErr
}
func (m *mockSequenceSession) Kill() error                 { return nil }
func (m *mockSequenceSession) Close() error                { return nil }
func (m *mockSequenceSession) Pid() int                    { return 1234 }
func (m *mockSequenceSession) CloseStdin() error           { return nil }

func TestSequenceHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_SEQUENCE") == "1" {
		main()
		return
	}
}

func TestSequence_NormalCompletion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := ptyx.SpawnOpts{
		Prog: os.Args[0],
		Args: []string{"-test.run=^TestSequenceHelperProcess$"},
		Env: append(os.Environ(),
			"GO_TEST_SEQUENCE=1",
			"PTYX_TEST_MODE=1",
		),
	}
	s, err := ptyx.Spawn(ctx, opts)
	if err != nil {
		t.Fatalf("failed to spawn command in pty: %v", err)
	}
	defer s.Close()

	var out bytes.Buffer
	readerDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&out, s.PtyReader())
		close(readerDone)
	}()

	waitDone := make(chan error, 1)
	go func() { waitDone <- s.Wait() }()

	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("command failed with error: %v\nOutput:\n%s", err, out.String())
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for normal completion; output:\n%s", out.String())
	}

	<-readerDone
	output := out.String()

	expected := []string{
		"[[PTYX_READY]]",
		"Loading...",
		"Process finished naturally.",
		"Process exited successfully with code 0.",
	}
	for _, sub := range expected {
		if !strings.Contains(output, sub) {
			t.Errorf("expected output to contain %q, but it didn't.\nFull output:\n%s", sub, output)
		}
	}
}

func TestSequence_Interruption(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := ptyx.SpawnOpts{
		Prog: os.Args[0],
		Args: []string{"-test.run=^TestSequenceHelperProcess$"},
		Env: append(os.Environ(),
			"GO_TEST_SEQUENCE=1",
			"PTYX_TEST_MODE=1",
		),
	}
	s, err := ptyx.Spawn(ctx, opts)
	if err != nil {
		t.Fatalf("failed to spawn command in pty: %v", err)
	}
	defer s.Close()

	ready := make(chan struct{})
	var out bytes.Buffer

	go func() {
		sc := bufio.NewScanner(io.TeeReader(s.PtyReader(), &out))
		sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
		for sc.Scan() {
			line := strings.TrimRight(sc.Text(), "\r")
			trim := strings.TrimSpace(line)

			if strings.Contains(trim, "[[PTYX_READY]]") || strings.Contains(trim, "Loading...") {
				select { case <-ready: default: close(ready) }
				cancel()
				return
			}
		}
	}()

	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for READY/Loading..., output:\n%s", out.String())
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- s.Wait() }()

	select {
	case waitErr := <-waitDone:
		if waitErr == nil {
			t.Fatalf("expected command to fail due to interruption, but it succeeded.\nOutput:\n%s", out.String())
		}
	case <-ctx.Done():
		waitErr := <-waitDone
		if waitErr == nil {
			t.Fatalf("expected failure after cancel, but got nil")
		}
	}

	if strings.Contains(out.String(), "[DEMO] Process finished naturally.") {
		t.Errorf("sequence should have been interrupted before finishing.\nOutput:\n%s", out.String())
	}
}

func TestRunCommandSequence(t *testing.T) {
	t.Run("Successful execution", func(t *testing.T) {
		ptyOutput := "[[PTYX_READY]]\nLoading...\n[[PTYX_DONE]]\n"
		eofChan := make(chan struct{})
		mockSess := &mockSequenceSession{
			ptyIn:   &bytes.Buffer{},
			ptyOut:  &eofSignalingReader{Reader: bytes.NewBufferString(ptyOutput), done: eofChan},
			waitErr: nil,
			eofChan: eofChan,
		}

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runCommandSequence(mockSess)

		w.Close()
		os.Stdout = oldStdout
		var capturedOut bytes.Buffer
		_, _ = io.Copy(&capturedOut, r)

		if err != nil {
			t.Errorf("runCommandSequence() returned an unexpected error: %v", err)
		}

		writtenCmd := mockSess.ptyIn.String()
		if !strings.Contains(writtenCmd, "exit 0") {
			t.Errorf("Expected command written to PTY to contain 'exit 0', got %q", writtenCmd)
		}

		capturedStr := capturedOut.String()
		if !strings.Contains(capturedStr, "[[PTYX_READY]]") {
			t.Errorf("Expected stdout to contain '[[PTYX_READY]]', got %q", capturedStr)
		}
		if !strings.Contains(capturedStr, "[[PTYX_DONE]]") {
			t.Errorf("Expected stdout to contain '[[PTYX_DONE]]', got %q", capturedStr)
		}
	})

	t.Run("Wait returns error", func(t *testing.T) {
		ptyOutput := "[[PTYX_READY]]\nLoading...\n[[PTYX_DONE]]\n"
		eofChan := make(chan struct{})
		expectedErr := errors.New("wait failed")
		mockSess := &mockSequenceSession{
			ptyIn:   &bytes.Buffer{},
			ptyOut:  &eofSignalingReader{Reader: bytes.NewBufferString(ptyOutput), done: eofChan},
			waitErr: expectedErr,
			eofChan: eofChan,
		}

		oldStdout := os.Stdout
		os.Stdout, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		defer func() { os.Stdout = oldStdout }()

		err := runCommandSequence(mockSess)

		if !errors.Is(err, expectedErr) {
			t.Errorf("runCommandSequence() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestMainFunction(t *testing.T) {
	if os.Getenv("PTYX_TEST_RUN_MAIN") == "1" {
		main()
		return
	}

	t.Run("Normal completion", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainFunction$")
		cmd.Env = append(os.Environ(),
			"PTYX_TEST_RUN_MAIN=1",
			"PTYX_TEST_MODE=1",
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("cmd.Run() failed with: %v\noutput:\n%s", err, string(output))
		}

		outStr := string(output)
		if !strings.Contains(outStr, "Process finished naturally") {
			t.Errorf("expected output to contain 'Process finished naturally', but it didn't. Got:\n%s", outStr)
		}
	})

	t.Run("Interrupted execution", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainFunction$")
		cmd.Env = append(os.Environ(), "PTYX_TEST_RUN_MAIN=1")

		output, err := cmd.CombinedOutput()

		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("cmd.Run() should have exited with code 1, but got err: %v\noutput:\n%s", err, string(output))
		}

		outStr := string(output)
		if !strings.Contains(outStr, "Process was interrupted") {
			t.Errorf("expected output to contain 'Process was interrupted', but it didn't. Got:\n%s", outStr)
		}
	})
}

func TestHandleWaitResult(t *testing.T) {
	t.Run("Successful completion", func(t *testing.T) {
		exitCode, output := handleWaitResult(context.Background(), nil)
		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", exitCode)
		}
		if !strings.Contains(output, "Process finished naturally") {
			t.Errorf("Expected output to contain 'Process finished naturally', but it didn't. Got:\n%s", output)
		}
		if !strings.Contains(output, "Process exited successfully with code 0") {
			t.Errorf("Expected output to contain 'Process exited successfully with code 0', but it didn't. Got:\n%s", output)
		}
	})

	t.Run("Generic error", func(t *testing.T) {
		testErr := errors.New("something went wrong")
		exitCode, output := handleWaitResult(context.Background(), testErr)
		if exitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", exitCode)
		}
		if !strings.Contains(output, "Go error: something went wrong") {
			t.Errorf("Expected output to contain error message, but it didn't. Got:\n%s", output)
		}
	})

	t.Run("ExitError without context cancellation", func(t *testing.T) {
		exitErr := &ptyx.ExitError{ExitCode: 127}
		exitCode, output := handleWaitResult(context.Background(), exitErr)
		if exitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", exitCode)
		}
		if !strings.Contains(output, "Exit code: 127") {
			t.Errorf("Expected output to contain 'Exit code: 127', but it didn't. Got:\n%s", output)
		}
		if strings.Contains(output, "Process was interrupted") {
			t.Errorf("Output should not contain 'Process was interrupted'. Got:\n%s", output)
		}
	})

	t.Run("ExitError with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		exitErr := &ptyx.ExitError{ExitCode: 1}
		exitCode, output := handleWaitResult(ctx, exitErr)
		if exitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", exitCode)
		}
		if !strings.Contains(output, "Exit code: 1") {
			t.Errorf("Expected output to contain 'Exit code: 1', but it didn't. Got:\n%s", output)
		}
		if !strings.Contains(output, "Process was interrupted") {
			t.Errorf("Expected output to contain 'Process was interrupted', but it didn't. Got:\n%s", output)
		}
	})
}

func TestRunCommandSequence_Errors(t *testing.T) {
	t.Run("Write to PTY fails", func(t *testing.T) {
		expectedErr := errors.New("write failed")
		mockSess := &mockSequenceSession{
			ptyIn:           &bytes.Buffer{},
			ptyOut:          bytes.NewBufferString(""),
			forceWriteError: expectedErr,
		}

		err := runCommandSequence(mockSess)

		if !errors.Is(err, expectedErr) {
			t.Errorf("runCommandSequence() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("PTY Read returns error", func(t *testing.T) {
		waitErr := errors.New("process exited")
		mockSess := &mockSequenceSession{
			ptyIn:   &bytes.Buffer{},
			ptyOut:  &errorReader{err: errors.New("read failed")},
			waitErr: waitErr,
		}

		oldStdout := os.Stdout
		os.Stdout, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		defer func() { os.Stdout = oldStdout }()

		err := runCommandSequence(mockSess)

		if !errors.Is(err, waitErr) {
			t.Errorf("runCommandSequence() should return the error from Wait(), got %v, want %v", err, waitErr)
		}
	})
}
