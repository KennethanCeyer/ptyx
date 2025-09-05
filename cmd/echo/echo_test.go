package main

import (
	"bytes"
	"strings"
	"testing"
)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			var out, errOut bytes.Buffer

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
