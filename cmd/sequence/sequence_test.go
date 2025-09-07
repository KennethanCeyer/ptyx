package main_test

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

var sequenceBinaryPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "sequence-test-")
	if err != nil {
		log.Fatalf("failed to create temp dir for test binary: %v", err)
	}

	binPath := filepath.Join(tmpDir, "sequence")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to build sequence binary: %v\nOutput:\n%s", err, string(output))
	}
	sequenceBinaryPath = binPath

	code := m.Run()

	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestSequence_NormalCompletion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, sequenceBinaryPath)
	cmd.Env = append(os.Environ(), "PTYX_TEST_MODE=1")

	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed with error: %v\nOutput:\n%s", err, string(outputBytes))
	}
	output := string(outputBytes)

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

	cmd := exec.CommandContext(ctx, sequenceBinaryPath)
	cmd.Env = append(os.Environ(), "PTYX_TEST_MODE=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	interruptReady := make(chan struct{})
	var out bytes.Buffer
	go func() {
		scanner := bufio.NewScanner(io.TeeReader(stdout, &out))
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Loading...") {
				select {
				case <-interruptReady:
				default:
					close(interruptReady)
				}
			}
		}
	}()

	select {
	case <-interruptReady:
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("timed out waiting for the 'Loading...' signal")
	}

	if runtime.GOOS == "windows" {
		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("failed to kill process: %v", err)
		}
	} else {
		if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
			t.Fatalf("failed to send interrupt signal: %v", err)
		}
	}

	waitErr := cmd.Wait()
	output := out.String()

	if waitErr == nil {
		t.Fatalf("expected command to fail due to interruption, but it succeeded.\nOutput:\n%s", output)
	}

	if strings.Contains(output, "[DEMO] Command sequence finished.") {
		t.Errorf("sequence should have been interrupted before finishing, but it looks like it completed.\nOutput:\n%s", output)
	}
}
