package main

import (
	"bytes"
	"strings"
	"testing"
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
