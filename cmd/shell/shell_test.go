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

func TestShell_HelperProcess(t *testing.T) {
	if os.Getenv("PTYX_SHELL_HELPER") == "1" {
		switch os.Getenv("TEST_MODE") {
		case "run_generic_error":
			runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
				return errors.New("mocked generic error")
			}
		case "run_exit_error":
			runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
				return &ptyx.ExitError{ExitCode: 96}
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

func TestShell_MainExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestShell_HelperProcess$")
		cmd.Env = append(os.Environ(), "PTYX_SHELL_HELPER=1", "TEST_MODE=success")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		if strings.Contains(string(output), "Error:") {
			t.Errorf("unexpected error message in output: %s", string(output))
		}
	})

	t.Run("generic error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestShell_HelperProcess$")
		cmd.Env = append(os.Environ(), "PTYX_SHELL_HELPER=1", "TEST_MODE=run_generic_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Error: mocked generic error") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})

	t.Run("exit error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestShell_HelperProcess$")
		cmd.Env = append(os.Environ(), "PTYX_SHELL_HELPER=1", "TEST_MODE=run_exit_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have exited with code 96, but it succeeded")
		}
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 96 {
			t.Fatalf("expected exit code 96, got err: %v", err)
		}
		if len(output) > 0 {
			t.Errorf("expected no output, got: %s", string(output))
		}
	})
}
