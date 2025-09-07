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


func TestParseRunOpts(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    ptyx.SpawnOpts
		wantErr bool
	}{
		{
			name:    "No arguments provided",
			args:    []string{},
			want:    ptyx.SpawnOpts{},
			wantErr: true,
		},
		{
			name:    "Program only without arguments",
			args:    []string{"ls"},
			want:    ptyx.SpawnOpts{Prog: "ls", Args: []string{}},
			wantErr: false,
		},
		{
			name:    "Program with multiple arguments",
			args:    []string{"ls", "-l", "-a"},
			want:    ptyx.SpawnOpts{Prog: "ls", Args: []string{"-l", "-a"}},
			wantErr: false,
		},
		{
			name: "With size flags",
			args: []string{"-cols", "100", "-rows", "30", "top"},
			want: ptyx.SpawnOpts{Prog: "top", Args: []string{}, Cols: 100, Rows: 30},
		},
		{
			name: "With directory flag",
			args: []string{"-dir", "/tmp", "pwd"},
			want: ptyx.SpawnOpts{Prog: "pwd", Args: []string{}, Dir: "/tmp"},
		},
		{
			name: "With single environment variable",
			args: []string{"-env", "FOO=bar", "env"},
			want: ptyx.SpawnOpts{Prog: "env", Args: []string{}, Env: []string{"FOO=bar"}},
		},
		{
			name: "With multiple environment variables",
			args: []string{"-env", "FOO=bar", "-env", "BAZ=qux", "env"},
			want: ptyx.SpawnOpts{Prog: "env", Args: []string{}, Env: []string{"FOO=bar", "BAZ=qux"}},
		},
		{
			name: "All flags combined with program and args",
			args: []string{"-cols", "120", "-rows", "40", "-dir", "/home/user", "-env", "TERM=xterm-256color", "vim", "file.txt"},
			want: ptyx.SpawnOpts{
				Prog: "vim",
				Args: []string{"file.txt"},
				Cols: 120,
				Rows: 40,
				Dir:  "/home/user",
				Env:  []string{"TERM=xterm-256color"},
			},
		},
		{
			name:    "Unknown flag provided",
			args:    []string{"-unknown-flag", "ls"},
			want:    ptyx.SpawnOpts{},
			wantErr: true,
		},
		{
			name:    "Flags provided but no program",
			args:    []string{"-cols", "80"},
			want:    ptyx.SpawnOpts{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRunOpts(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRunOpts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRunOpts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRun_HelperProcess(t *testing.T) {
	if os.Getenv("PTYX_RUN_CMD_HELPER") != "1" {
		return
	}

	switch os.Getenv("TEST_MODE") {
	case "parse_error":
		parseRunOptsFunc = func(argv []string) (ptyx.SpawnOpts, error) {
			return ptyx.SpawnOpts{}, errors.New("mocked parse error")
		}
	case "run_error":
		parseRunOptsFunc = func(argv []string) (ptyx.SpawnOpts, error) {
			return ptyx.SpawnOpts{Prog: "sh"}, nil
		}
		runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
			return errors.New("mocked run error")
		}
	default:
		parseRunOptsFunc = func(argv []string) (ptyx.SpawnOpts, error) {
			return ptyx.SpawnOpts{Prog: "sh"}, nil
		}
		runInteractiveFunc = func(ctx context.Context, opts ptyx.SpawnOpts) error {
			return nil
		}
	}
	main()
	os.Exit(0)
}

func TestRun_MainExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRun_HelperProcess$")
		cmd.Env = append(os.Environ(), "PTYX_RUN_CMD_HELPER=1", "TEST_MODE=success")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process should have succeeded, but failed: %v\noutput: %s", err, string(output))
		}
		if len(output) > 0 {
			t.Errorf("expected no output, got: %s", string(output))
		}
	})

	t.Run("parse error", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRun_HelperProcess$")
		cmd.Env = append(os.Environ(), "PTYX_RUN_CMD_HELPER=1", "TEST_MODE=parse_error")
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
		cmd := exec.Command(os.Args[0], "-test.run=^TestRun_HelperProcess$")
		cmd.Env = append(os.Environ(), "PTYX_RUN_CMD_HELPER=1", "TEST_MODE=run_error")
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
