//go:build unix

package main

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

func TestHandleWaitResult_WithRealSignal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s, err := ptyx.Spawn(ctx, ptyx.SpawnOpts{Prog: "sleep", Args: []string{"10"}})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skipf("could not find 'sleep', skipping test: %v", err)
		}
		t.Fatalf("Failed to spawn 'sleep': %v", err)
	}
	defer s.Close()

	if err := s.Kill(); err != nil {
		t.Fatalf("Failed to kill process: %v", err)
	}

	waitErr := s.Wait()
	_, output := handleWaitResult(context.Background(), waitErr)
	if !strings.Contains(output, "Terminated by signal") {
		t.Errorf("Expected output to contain 'Terminated by signal', but it didn't. Got:\n%s", output)
	}
}
