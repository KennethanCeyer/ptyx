package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/KennethanCeyer/ptyx"
)


func TestParseResizeOpts(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    ptyx.SpawnOpts
		wantErr bool
	}{
		{
			name:    "No arguments",
			args:    []string{},
			want:    ptyx.SpawnOpts{},
			wantErr: true,
		},
		{
			name:    "Program only",
			args:    []string{"sh"},
			want:    ptyx.SpawnOpts{Prog: "sh", Args: []string{}},
			wantErr: false,
		},
		{
			name:    "Program with arguments",
			args:    []string{"bash", "-c", "echo hello"},
			want:    ptyx.SpawnOpts{Prog: "bash", Args: []string{"-c", "echo hello"}},
			wantErr: false,
		},
		{
			name:    "With size flags",
			args:    []string{"-cols", "100", "-rows", "30", "top"},
			want:    ptyx.SpawnOpts{Prog: "top", Args: []string{}, Cols: 100, Rows: 30},
			wantErr: false,
		},
		{
			name:    "Unknown flag",
			args:    []string{"-unknown", "sh"},
			want:    ptyx.SpawnOpts{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResizeOpts(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResizeOpts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseResizeOpts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMainHelper(t *testing.T) {
	if os.Getenv("PTYX_RESIZE_HELPER") != "1" {
		return
	}

	args := []string{os.Args[0]}
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-test.") {
			args = append(args, arg)
		}
	}
	os.Args = args

	switch os.Getenv("TEST_MODE") {
	case "parse_error":
		parseResizeOptsFunc = func(argv []string) (ptyx.SpawnOpts, error) {
			return ptyx.SpawnOpts{}, errors.New("mocked parse error")
		}
	case "run_error":
		parseResizeOptsFunc = func(argv []string) (ptyx.SpawnOpts, error) {
			return ptyx.SpawnOpts{Prog: "sh"}, nil
		}
		runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
			return errors.New("mocked run error")
		}
	default:
		parseResizeOptsFunc = func(argv []string) (ptyx.SpawnOpts, error) {
			return ptyx.SpawnOpts{Prog: "sh"}, nil
		}
		runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
			return nil
		}
	}
	main()
	os.Exit(0)
}

func TestMainExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_RESIZE_HELPER=1", "TEST_MODE=success")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		if len(output) > 0 {
			t.Errorf("expected no output, got: %s", string(output))
		}
	})

	t.Run("parse error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_RESIZE_HELPER=1", "TEST_MODE=parse_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		outStr := string(output)
		if !strings.Contains(outStr, "mocked parse error") {
			t.Errorf("expected parse error message, got: %s", outStr)
		}
	})

	t.Run("run error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelper$")
		cmd.Env = append(os.Environ(), "PTYX_RESIZE_HELPER=1", "TEST_MODE=run_error")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("process should have failed, but it succeeded")
		}
		outStr := string(output)
		if !strings.Contains(outStr, "Error: mocked run error") {
			t.Errorf("expected run error message, got: %s", outStr)
		}
	})
}
