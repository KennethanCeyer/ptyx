package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/KennethanCeyer/ptyx"
	"github.com/KennethanCeyer/ptyx/testptyx"
)

func TestProcessStream(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Simple text",
			input: "hello world",
			want:  "[EVENT:TEXT] \"hello world\"\n",
		},
		{
			name:  "Text with newline",
			input: "hello\nworld",
			want:  "[EVENT:TEXT] \"hello\"\n[EVENT:CONTROL] \"\\n\"\n[EVENT:TEXT] \"world\"\n",
		},
		{
			name:  "ANSI escape sequence",
			input: "text\x1b[31mred\x1b[0m",
			want:  "[EVENT:TEXT] \"text\"\n[EVENT:ANSI] \"31m\"\n[EVENT:TEXT] \"red\"\n[EVENT:ANSI] \"0m\"\n",
		},
		{
			name:  "Mixed content",
			input: "A\r\n\x1b[?25lC",
			want:  "[EVENT:TEXT] \"A\"\n[EVENT:CONTROL] \"\\r\"\n[EVENT:CONTROL] \"\\n\"\n[EVENT:ANSI] \"?25l\"\n[EVENT:TEXT] \"C\"\n",
		},
		{
			name:  "Incomplete CSI at EOF",
			input: "text\x1b[1;31",
			want:  "[EVENT:TEXT] \"text\"\n[EVENT:ANSI] \"1;31\"\n",
		},
		{
			name:  "Non-CSI escape sequence",
			input: "\x1b]",
			want:  "[EVENT:UNHANDLED] \"]\"\n",
		},
		{
			name:  "Escape at EOF",
			input: "hello\x1b",
			want:  "[EVENT:TEXT] \"hello\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			var out bytes.Buffer
			processStream(&out, in)
			if got := out.String(); got != tt.want {
				t.Errorf("processStream() output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMainHelper(t *testing.T) {
	if os.Getenv("PTYX_EVENT_HELPER") != "1" {
		return
	}

	switch os.Getenv("TEST_MODE") {
	case "caller_error":
		runtimeCallerFunc = func(skip int) (pc uintptr, file string, line int, ok bool) {
			return 0, "", 0, false
		}
	case "newConsole_error":
		newConsoleFunc = func() (ptyx.Console, error) {
			return nil, errors.New("mocked newConsole error")
		}
	case "spawn_error":
		newConsoleFunc = func() (ptyx.Console, error) {
			return testptyx.NewMockConsole(""), nil
		}
		ptyxSpawnFunc = func(ctx context.Context, opts ptyx.SpawnOpts) (ptyx.Session, error) {
			return nil, errors.New("mocked spawn error")
		}
	case "wait_error":
		newConsoleFunc = func() (ptyx.Console, error) {
			return testptyx.NewMockConsole(""), nil
		}
		mockSession := testptyx.NewMockSession("")
		mockSession.WaitError = errors.New("mocked wait error")
		ptyxSpawnFunc = func(ctx context.Context, opts ptyx.SpawnOpts) (ptyx.Session, error) {
			return mockSession, nil
		}
	default:
		newConsoleFunc = func() (ptyx.Console, error) {
			return testptyx.NewMockConsole(""), nil
		}
		mockSession := testptyx.NewMockSession("some output")
		ptyxSpawnFunc = func(ctx context.Context, opts ptyx.SpawnOpts) (ptyx.Session, error) {
			return mockSession, nil
		}
	}
	main()
	os.Exit(0)
}

func TestMainExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_EVENT_HELPER=1", "TEST_MODE=success")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		outStr := string(output)
		if !strings.Contains(outStr, "Event stream terminated") {
			t.Errorf("expected success message, got: %s", outStr)
		}
	})

	t.Run("caller error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_EVENT_HELPER=1", "TEST_MODE=caller_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Error: cannot determine project root") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})

	t.Run("newConsole error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_EVENT_HELPER=1", "TEST_MODE=newConsole_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Error creating console: mocked newConsole error") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})

	t.Run("spawn error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_EVENT_HELPER=1", "TEST_MODE=spawn_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		if !strings.Contains(string(output), "Error spawning process: mocked spawn error") {
			t.Errorf("expected error message, got: %s", string(output))
		}
	})
}
