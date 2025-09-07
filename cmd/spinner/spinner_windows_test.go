//go:build windows

package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
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

	waitErr := s.Wait()

	s.Close()

	select {
	case <-readerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY reader to finish")
	}

	if waitErr != nil {
		t.Fatalf("Subprocess exited with an unexpected error: %v\nOutput:\n%s",
			waitErr, out.String())
	}

	output := out.String()
	if !strings.Contains(output, "Working...") {
		t.Errorf("Subprocess did not produce expected stdout. Got: %s", output)
	}
	if !strings.Contains(output, "Done.") {
		t.Errorf("Subprocess did not print done message. Got: %s", output)
	}
}
