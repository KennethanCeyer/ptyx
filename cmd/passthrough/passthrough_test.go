package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/KennethanCeyer/ptyx"
)

func TestMainHelper(t *testing.T) {
	if os.Getenv("PTYX_PASSTHROUGH_HELPER") == "1" {
		switch os.Getenv("TEST_MODE") {
		case "runtimeCaller_error":
			runtimeCallerFunc = func(skip int) (pc uintptr, file string, line int, ok bool) {
				return 0, "", 0, false
			}
		case "runInteractive_generic_error":
			runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
				return errors.New("mocked generic error")
			}
		case "runInteractive_exit_error_nonzero":
			runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
				return &ptyx.ExitError{ExitCode: 96}
			}
		case "runInteractive_exit_error_zero":
			runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
				return &ptyx.ExitError{ExitCode: 0}
			}
		default:
			runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
				return nil
			}
		}
		main()
		os.Exit(0)
	}
}

func TestMainExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_PASSTHROUGH_HELPER=1", "TEST_MODE=success")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		if strings.Contains(string(output), "Error:") {
			t.Errorf("unexpected error message in output: %s", string(output))
		}
	})

	t.Run("runtimeCaller error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_PASSTHROUGH_HELPER=1", "TEST_MODE=runtimeCaller_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Error: cannot determine project root") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})

	t.Run("runInteractive generic error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_PASSTHROUGH_HELPER=1", "TEST_MODE=runInteractive_generic_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Error: mocked generic error") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})

	t.Run("runInteractive exit error non-zero", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_PASSTHROUGH_HELPER=1", "TEST_MODE=runInteractive_exit_error_nonzero")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Error: process exited with status 96") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})

	t.Run("runInteractive exit error zero", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_PASSTHROUGH_HELPER=1", "TEST_MODE=runInteractive_exit_error_zero")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		if strings.Contains(string(output), "Error:") {
			t.Errorf("unexpected error message in output: %s", string(output))
		}
	})
}
