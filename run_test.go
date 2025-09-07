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
	if runtime.GOOS == "windows" {
		// TODO: I/O 리다이렉션 문제로 인해 Windows에서 비활성화합니다.
		// 추후 Windows 케이스를 다시 활성화할 예정입니다.
		t.Skip("Skipping non-console interactive test on Windows; will be covered in the future.")
	}

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
		// TODO: 중첩된 PTY 환경에서 발생하는 데드락 문제로 인해 Windows에서 비활성화합니다.
		// 추후 Windows 케이스를 다시 활성화할 예정입니다.
		t.Skip("Skipping nested PTY test on Windows; will be covered in the future.")
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

func TestRunInteractive_ErrorPaths(t *testing.T) {
	baseOpts := SpawnOpts{
		Prog: os.Args[0],
		Args: []string{"-test.run=^TestRunHelperProcess$"},
		Env:  append(os.Environ(), "PTYX_RUN_HELPER=1"),
	}

	t.Run("NewConsoleError", func(t *testing.T) {
		originalNewConsole := newConsoleFunc
		newConsoleFunc = func() (Console, error) {
			return nil, errors.New("mock new console error")
		}
		t.Cleanup(func() { newConsoleFunc = originalNewConsole })

		err := RunInteractive(context.Background(), baseOpts)
		if err == nil {
			t.Fatal("RunInteractive should have failed but did not")
		}
		if !strings.Contains(err.Error(), "mock new console error") {
			t.Errorf("Expected 'mock new console error', got %v", err)
		}
	})

	t.Run("NonConsole_SpawnError", func(t *testing.T) {
		originalNewConsole := newConsoleFunc
		newConsoleFunc = func() (Console, error) {
			return nil, ErrNotAConsole
		}
		t.Cleanup(func() { newConsoleFunc = originalNewConsole })

		originalSpawn := spawnFunc
		spawnFunc = func(ctx context.Context, opts SpawnOpts) (Session, error) {
			return nil, errors.New("mock spawn error")
		}
		t.Cleanup(func() { spawnFunc = originalSpawn })

		err := RunInteractive(context.Background(), baseOpts)
		if err == nil {
			t.Fatal("RunInteractive should have failed but did not")
		}
		if !strings.Contains(err.Error(), "mock spawn error") {
			t.Errorf("Expected 'mock spawn error', got %v", err)
		}
	})

	t.Run("NonConsole_ContextCancel", func(t *testing.T) {
		originalNewConsole := newConsoleFunc
		newConsoleFunc = func() (Console, error) {
			return nil, ErrNotAConsole
		}
		t.Cleanup(func() { newConsoleFunc = originalNewConsole })

		mockSess := newMockSession("")
		waitCh := make(chan struct{})
		mockSess.waitFunc = func() error {
			<-waitCh
			return context.Canceled
		}
		mockSess.closeFunc = func() error {
			close(waitCh)
			return nil
		}
		originalSpawn := spawnFunc
		spawnFunc = func(ctx context.Context, opts SpawnOpts) (Session, error) {
			return mockSess, nil
		}
		t.Cleanup(func() { spawnFunc = originalSpawn })

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := RunInteractive(ctx, baseOpts)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("RunInteractive() error = %v, want context.Canceled", err)
		}
	})

	t.Run("Console_SpawnError", func(t *testing.T) {
		originalNewConsole := newConsoleFunc
		newConsoleFunc = func() (Console, error) {
			return newMockConsole(""), nil
		}
		t.Cleanup(func() { newConsoleFunc = originalNewConsole })

		originalSpawn := spawnFunc
		spawnFunc = func(ctx context.Context, opts SpawnOpts) (Session, error) {
			return nil, errors.New("mock spawn error")
		}
		t.Cleanup(func() { spawnFunc = originalSpawn })

		err := RunInteractive(context.Background(), baseOpts)
		if err == nil {
			t.Fatal("RunInteractive should have failed but did not")
		}
		if !strings.Contains(err.Error(), "mock spawn error") {
			t.Errorf("Expected 'mock spawn error', got %v", err)
		}
	})

	t.Run("Console_MuxStartError", func(t *testing.T) {
		originalNewConsole := newConsoleFunc
		newConsoleFunc = func() (Console, error) {
			return newMockConsole(""), nil
		}
		t.Cleanup(func() { newConsoleFunc = originalNewConsole })

		originalSpawn := spawnFunc
		spawnFunc = func(ctx context.Context, opts SpawnOpts) (Session, error) {
			return newMockSession(""), nil
		}
		t.Cleanup(func() { spawnFunc = originalSpawn })

		originalNewMux := newMuxFunc
		newMuxFunc = func() Mux {
			return &mockMux{startErr: errors.New("mock mux start error")}
		}
		t.Cleanup(func() { newMuxFunc = originalNewMux })

		err := RunInteractive(context.Background(), baseOpts)
		if err == nil {
			t.Fatal("RunInteractive should have failed but did not")
		}
		if !strings.Contains(err.Error(), "mock mux start error") {
			t.Errorf("Expected 'mock mux start error', got %v", err)
		}
	})

	t.Run("Console_ExitError", func(t *testing.T) {
		originalNewConsole := newConsoleFunc
		newConsoleFunc = func() (Console, error) {
			return newMockConsole(""), nil
		}
		t.Cleanup(func() { newConsoleFunc = originalNewConsole })

		mockSess := newMockSession("")
		mockSess.waitFunc = func() error {
			return &ExitError{ExitCode: 42}
		}
		originalSpawn := spawnFunc
		spawnFunc = func(ctx context.Context, opts SpawnOpts) (Session, error) {
			return mockSess, nil
		}
		t.Cleanup(func() { spawnFunc = originalSpawn })

		err := RunInteractive(context.Background(), baseOpts)
		var exitErr *ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode != 42 {
			t.Fatalf("RunInteractive() error = %v (type %T), want *ptyx.ExitError with code 42", err, err)
		}
	})
}
