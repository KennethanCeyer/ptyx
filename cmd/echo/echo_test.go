package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/KennethanCeyer/ptyx"
	"github.com/KennethanCeyer/ptyx/testptyx"
)

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) { return 0, errors.New("read failed") }

func TestEchoLoop(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOut string
		wantErr string
	}{
		{
			name:    "Simple input",
			input:   "hello",
			wantOut: "read 5 bytes: \"hello\"\r\n",
			wantErr: "",
		},
		{
			name:    "Ctrl+C terminates",
			input:   "abc\x03def",
			wantOut: "read 3 bytes: \"abc\"\r\n",
			wantErr: "",
		},
		{
			name:    "Ctrl+C as first char",
			input:   "\x03def",
			wantOut: "",
			wantErr: "",
		},
		{
			name:    "Empty input (EOF)",
			input:   "",
			wantOut: "",
			wantErr: "",
		},
		{
			name:    "Input is exact buffer size multiple",
			input:   strings.Repeat("a", 1024),
			wantOut: "read 1024 bytes: \"" + strings.Repeat("a", 1024) + "\"\r\n",
			wantErr: "",
		},
		{
			name:    "Read error",
			input:   "ignored",
			wantOut: "",
			wantErr: "read error: read failed\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var in io.Reader = strings.NewReader(tt.input)
			var out, errOut bytes.Buffer

			if tt.name == "Read error" {
				in = &errorReader{}
			}
			echoLoop(in, &out, &errOut)

			if gotOut := out.String(); gotOut != tt.wantOut {
				t.Errorf("echoLoop() output = %q, want %q", gotOut, tt.wantOut)
			}
			if gotErr := errOut.String(); gotErr != tt.wantErr {
				t.Errorf("echoLoop() error output = %q, want %q", gotErr, tt.wantErr)
			}
		})
	}
}

func TestEcho_HelperProcess(t *testing.T) {
	if os.Getenv("PTYX_ECHO_HELPER") != "1" {
		return
	}

	var mockConsole *testptyx.MockConsole
	switch os.Getenv("TEST_MODE") {
	case "newConsole_error":
		newConsoleFunc = func() (ptyx.Console, error) {
			return nil, errors.New("mocked newConsole error")
		}
	case "makeRaw_error":
		mockConsole = testptyx.NewMockConsole("ignored")
		mockConsole.MakeRawError = errors.New("mocked makeRaw error")
		newConsoleFunc = func() (ptyx.Console, error) {
			return mockConsole, nil
		}
	default:
		mockConsole = testptyx.NewMockConsole("test\x03")
		newConsoleFunc = func() (ptyx.Console, error) {
			return mockConsole, nil
		}
	}
	main()
	if mockConsole != nil {
		fmt.Print(mockConsole.OutBuffer.String())
	}
	os.Exit(0)
}

func TestMainExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestEcho_HelperProcess$")
		cmd.Env = append(os.Environ(), "PTYX_ECHO_HELPER=1", "TEST_MODE=success")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		outStr := string(output)
		if !strings.Contains(outStr, "Entering raw echo mode.") {
			t.Errorf("expected success message, got: %s", outStr)
		}
		if !strings.Contains(outStr, "read 4 bytes: \"test\"") {
			t.Errorf("expected echo output, got: %s", outStr)
		}
	})

	t.Run("newConsole error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestEcho_HelperProcess$")
		cmd.Env = append(os.Environ(), "PTYX_ECHO_HELPER=1", "TEST_MODE=newConsole_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		outStr := string(output)
		if !strings.Contains(outStr, "failed to create console: mocked newConsole error") {
			t.Errorf("expected error message, got: %s", outStr)
		}
	})

	t.Run("makeRaw error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestEcho_HelperProcess$")
		cmd.Env = append(os.Environ(), "PTYX_ECHO_HELPER=1", "TEST_MODE=makeRaw_error")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		outStr := string(output)
		if !strings.Contains(outStr, "raw mode error: mocked makeRaw error") {
			t.Errorf("expected makeRaw error message, got: %s", outStr)
		}
	})
}
