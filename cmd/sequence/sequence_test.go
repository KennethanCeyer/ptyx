package main_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSequence_NormalCompletion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", ".")
	cmd.Env = append(os.Environ(), "PTYX_TEST_MODE=1")

	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed with error: %v\nOutput:\n%s", err, string(outputBytes))
	}
	output := string(outputBytes)

	if err != nil {
		t.Fatalf("command failed with error: %v\nOutput:\n%s", err, output)
	}

	expectedSubstrings := []string{
		"Loading...",
		"Process finished naturally.",
		"Process exited successfully with code 0.",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("expected output to contain %q, but it didn't.\nFull output:\n%s", sub, output)
		}
	}
}

func TestSequence_Interruption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", ".")
	cmd.Env = append(os.Environ(), "PTYX_TEST_MODE=1")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if runtime.GOOS == "windows" {
		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("failed to kill process: %v", err)
		}
	} else {
		if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
			t.Fatalf("failed to send interrupt signal: %v", err)
		}
	}

	err := cmd.Wait()
	output := out.String()

	if err == nil {
		t.Fatalf("expected command to fail due to interruption, but it succeeded.\nOutput:\n%s", output)
	}

	if strings.Contains(output, "[DEMO] Command sequence finished.") {
		t.Errorf("sequence should have been interrupted before finishing, but it looks like it completed.\nOutput:\n%s", output)
	}
}
