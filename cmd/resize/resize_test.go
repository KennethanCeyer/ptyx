package main

import (
	"reflect"
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
