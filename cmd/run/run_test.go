package main

import (
	"reflect"
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
