//go:build unix

package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

func TestSpinnerMain(t *testing.T) {
	if os.Getenv("GO_TEST_SPINNER") == "1" {
		main()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s, err := ptyx.Spawn(ctx, ptyx.SpawnOpts{
		Prog: os.Args[0],
		Args: []string{"-test.run=^TestSpinnerMain$"},
		Env:  append(os.Environ(), "GO_TEST_SPINNER=1"),
	})
	if err != nil {
		t.Fatalf("Failed to spawn subprocess in PTY: %v", err)
	}
	defer s.Close()

	var out bytes.Buffer
	readerDone := make(chan struct{})
	go func() {
		io.Copy(&out, s.PtyReader())
		close(readerDone)
	}()

	time.Sleep(200 * time.Millisecond)

	proc, err := os.FindProcess(s.Pid())
	if err != nil {
		t.Fatalf("Failed to find process with PID %d: %v", s.Pid(), err)
	}
	if err := proc.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("Failed to send SIGINT to subprocess: %v", err)
	}

	waitErr := s.Wait()

	select {
	case <-readerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY reader to finish")
	}

	if waitErr != nil {
		t.Fatalf("Subprocess exited with an unexpected error after SIGINT: %v\nOutput:\n%s",
			waitErr, out.String())
	}

	output := out.String()
	if !strings.Contains(output, "Working...") {
		t.Errorf("Subprocess did not produce expected stdout. Got: %s", output)
	}
	if !strings.Contains(output, "Spinner stopped.") {
		t.Errorf("Subprocess did not print stop message. Got: %s", output)
	}
}
