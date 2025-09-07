package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KennethanCeyer/ptyx"
)

func TestGetProjectRoot(t *testing.T) {
	root, err := getProjectRoot()
	if err != nil {
		t.Fatalf("getProjectRoot() failed: %v", err)
	}

	if root == "" {
		t.Error("getProjectRoot() returned an empty string")
	}

	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Errorf("go.mod not found in the determined project root: %s", root)
	}
}

func TestGetProjectRoot_CallerFails(t *testing.T) {
	originalCaller := runtimeCaller
	defer func() { runtimeCaller = originalCaller }()

	runtimeCaller = func(skip int) (pc uintptr, file string, line int, ok bool) {
		return 0, "", 0, false
	}

	_, err := getProjectRoot()
	if err == nil {
		t.Fatal("getProjectRoot should have failed but it didn't")
	}
	expectedErr := "cannot determine project root: runtime.Caller failed"
	if err.Error() != expectedErr {
		t.Errorf("getProjectRoot() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestMainHelper(t *testing.T) {
	if os.Getenv("PTYX_NESTED_HELPER") == "1" {
		switch os.Getenv("TEST_MODE") {
		case "getProjectRoot_error":
			getProjectRootFunc = func() (string, error) {
				return "", errors.New("mocked getProjectRoot error")
			}
		case "runInteractive_error":
			getProjectRootFunc = func() (string, error) {
				return "/tmp", nil
			}
			runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
				return errors.New("mocked runInteractive error")
			}
		default:
			getProjectRootFunc = func() (string, error) {
				return "/tmp", nil
			}
			runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
				if opts.Prog != "go" || opts.Dir != "/tmp" {
					return fmt.Errorf("unexpected opts: %+v", opts)
				}
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
		cmd.Env = append(os.Environ(), "PTYX_NESTED_HELPER=1", "TEST_MODE=success")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		outStr := string(output)
		if !strings.Contains(outStr, "--- Running 'go run ./cmd/shell' in a PTY (nested PTY) ---") {
			t.Errorf("expected success message, got: %s", outStr)
		}
		if strings.Contains(outStr, "Error:") {
			t.Errorf("unexpected error message in output: %s", outStr)
		}
	})

	t.Run("getProjectRoot error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_NESTED_HELPER=1", "TEST_MODE=getProjectRoot_error")
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Fatalf("process should have failed due to os.Exit(1), but it succeeded. output: %s", string(output))
		}
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got err: %v", err)
		}

		outStr := string(output)
		if !strings.Contains(outStr, "mocked getProjectRoot error") {
			t.Errorf("expected error message in stderr, got: %s", outStr)
		}
	})

	t.Run("runInteractive error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_NESTED_HELPER=1", "TEST_MODE=runInteractive_error")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		outStr := string(output)
		if !strings.Contains(outStr, "Error: mocked runInteractive error") {
			t.Errorf("expected error message in output, got: %s", outStr)
		}
	})
}
