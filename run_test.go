package ptyx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunHelperProcess(t *testing.T) {
	if os.Getenv("PTYX_RUN_HELPER") != "1" {
		return
	}
	switch os.Getenv("MODE") {
	case "sleep":
		time.Sleep(5 * time.Second)
	case "exit96":
		os.Exit(96)
	default:
		os.Exit(0)
	}
}

func TestRun(t *testing.T) {
	baseOpts := SpawnOpts{
		Prog: os.Args[0],
		Args: []string{"-test.run=^TestRunHelperProcess$"},
		Env:  append(os.Environ(), "PTYX_RUN_HELPER=1"),
	}

	t.Run("Success", func(t *testing.T) {
		opts := baseOpts
		opts.Env = append(opts.Env, "MODE=success")
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("Run() with successful command failed: %v", err)
		}
	})

	t.Run("ExitError", func(t *testing.T) {
		opts := baseOpts
		opts.Env = append(opts.Env, "MODE=exit96")
		err := Run(context.Background(), opts)
		if err == nil {
			t.Fatal("Run() with failing command should have returned an error")
		}
		var exitErr *ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode != 96 {
			t.Fatalf("Run() error = %v (type %T), want *ptyx.ExitError with code 96", err, err)
		}
	})

	t.Run("ContextCancel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		opts := baseOpts
		opts.Env = append(opts.Env, "MODE=sleep")
		err := Run(ctx, opts)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Run() error = %v, want context.DeadlineExceeded", err)
		}
	})
}

func TestRunInteractiveHelperProcess(t *testing.T) {
	if os.Getenv("PTYX_INTERACTIVE_HELPER") != "1" {
		return
	}
	os.Stdout.WriteString("helper process ran")
	os.Exit(0)
}

func TestRunInteractive_NonConsole(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := SpawnOpts{
		Prog: os.Args[0],
		Args: []string{"-test.run=^TestRunInteractiveHelperProcess$"},
		Env:  append(os.Environ(), "PTYX_INTERACTIVE_HELPER=1"),
	}

	err := RunInteractive(context.Background(), opts)

	w.Close()
	os.Stdout = oldStdout
	var out bytes.Buffer
	io.Copy(&out, r)

	if err != nil {
		t.Fatalf("RunInteractive (non-console) failed: %v", err)
	}

	if !strings.Contains(out.String(), "helper process ran") {
		t.Errorf("Expected output from helper process, but got: %s", out.String())
	}
}

func TestRunInteractive_Console(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping nested PTY test on Windows due to platform I/O limitations causing deadlocks.")
	}

	if os.Getenv("PTYX_INTERACTIVE_CONSOLE_TEST") == "1" {
		opts := SpawnOpts{
			Prog: os.Args[0],
			Args: []string{"-test.run=^TestRunInteractiveHelperProcess$"},
			Env:  append(os.Environ(), "PTYX_INTERACTIVE_HELPER=1"),
		}
		err := RunInteractive(context.Background(), opts)
		if err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	t.Run("in_pty", func(t *testing.T) {
		opts := SpawnOpts{
			Prog: os.Args[0],
			Args: []string{"-test.run=^TestRunInteractive_Console$"},
			Env:  append(os.Environ(), "PTYX_INTERACTIVE_CONSOLE_TEST=1"),
		}

		s, err := Spawn(context.Background(), opts)
		if err != nil {
			t.Fatalf("Spawn for interactive test failed: %v", err)
		}
		defer s.Close()

		go io.Copy(io.Discard, s.PtyReader())

		s.CloseStdin()

		err = s.Wait()

		if !isExpectedWaitErrorAfterPTYClose(err) {
			t.Fatalf("RunInteractive (console) failed with an unexpected error: %v", err)
		}
	})
}
