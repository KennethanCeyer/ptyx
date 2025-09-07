package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/KennethanCeyer/ptyx"
	"github.com/KennethanCeyer/ptyx/testptyx"
)

type mockReader struct {
	chunks [][]byte
	idx    int
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	if r.idx >= len(r.chunks) {
		return 0, io.EOF
	}
	n = copy(p, r.chunks[r.idx])
	r.idx++
	return n, nil
}

func TestPromptDetector(t *testing.T) {
	prompt := []byte("prompt: ")

	t.Run("PromptInSingleRead", func(t *testing.T) {
		input := bytes.NewReader([]byte("some data before prompt: and after"))
		promptFound := make(chan struct{})
		detector := &promptDetector{
			r:           input,
			prompt:      prompt,
			promptFound: promptFound,
		}
		go io.Copy(io.Discard, detector)

		select {
		case <-promptFound:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for prompt")
		}
	})

	t.Run("PromptSplitAcrossReads", func(t *testing.T) {
		mockIn := &mockReader{
			chunks: [][]byte{
				[]byte("some data before pro"),
				[]byte("mpt: and after"),
			},
		}
		promptFound := make(chan struct{})
		detector := &promptDetector{
			r:           mockIn,
			prompt:      prompt,
			promptFound: promptFound,
		}
		go io.Copy(io.Discard, detector)

		select {
		case <-promptFound:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for prompt")
		}
	})

	t.Run("NoPrompt", func(t *testing.T) {
		input := bytes.NewReader([]byte("some other data"))
		promptFound := make(chan struct{})
		detector := &promptDetector{r: input, prompt: prompt, promptFound: promptFound}
		go io.Copy(io.Discard, detector)

		select {
		case <-promptFound:
			t.Fatal("prompt was found but should not have been")
		case <-time.After(50 * time.Millisecond):
		}
	})
}

func TestMainHelper(t *testing.T) {
	if os.Getenv("PTYX_SCAN_HELPER") == "1" {
		originalStdout := stdout
		stdout = io.Discard
		defer func() { stdout = originalStdout }()

		switch os.Getenv("TEST_MODE") {
		case "caller_error":
			runtimeCallerFunc = func(skip int) (pc uintptr, file string, line int, ok bool) {
				return 0, "", 0, false
			}
		case "spawn_error":
			ptyxSpawnFunc = func(ctx context.Context, opts ptyx.SpawnOpts) (ptyx.Session, error) {
				return nil, errors.New("mocked spawn error")
			}
		case "timeout_error":
			mockSession := testptyx.NewMockSession("some other output")
			ptyxSpawnFunc = func(ctx context.Context, opts ptyx.SpawnOpts) (ptyx.Session, error) {
				return mockSession, nil
			}
			closedChan := make(chan time.Time)
			close(closedChan)
			timeAfterFunc = func(d time.Duration) <-chan time.Time {
				return closedChan
			}
		case "write_error":
			mockSession := testptyx.NewMockSession("What is your name? ")
			mockSession.ForceWriteError = errors.New("mocked write error")
			ptyxSpawnFunc = func(ctx context.Context, opts ptyx.SpawnOpts) (ptyx.Session, error) {
				return mockSession, nil
			}
		default:
			mockSession := testptyx.NewMockSession("What is your name? Hello, World!")
			ptyxSpawnFunc = func(ctx context.Context, opts ptyx.SpawnOpts) (ptyx.Session, error) {
				return mockSession, nil
			}
		}
		main()
		os.Exit(0)
	}
}

func TestMainExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_SCAN_HELPER=1", "TEST_MODE=success")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		if strings.Contains(string(output), "Error:") {
			t.Errorf("unexpected error message in output: %s", string(output))
		}
	})

	t.Run("caller error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_SCAN_HELPER=1", "TEST_MODE=caller_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Error: cannot determine project root") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})

	t.Run("spawn error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_SCAN_HELPER=1", "TEST_MODE=spawn_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Failed to spawn: mocked spawn error") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})

	t.Run("timeout error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_SCAN_HELPER=1", "TEST_MODE=timeout_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Timeout: Did not find expected prompt in PTY output.") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})

	t.Run("write error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_SCAN_HELPER=1", "TEST_MODE=write_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Failed to write to PTY: mocked write error") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})
}
