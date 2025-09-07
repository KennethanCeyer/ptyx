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
			name:    "No arguments",
			args:    []string{},
			want:    ptyx.SpawnOpts{},
			wantErr: true,
		},
		{
			name:    "Program only",
			args:    []string{"ls"},
			want:    ptyx.SpawnOpts{Prog: "ls", Args: []string{}},
			wantErr: false,
		},
		{
			name:    "Program with arguments",
			args:    []string{"ls", "-l", "-a"},
			want:    ptyx.SpawnOpts{Prog: "ls", Args: []string{"-l", "-a"}},
			wantErr: false,
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
