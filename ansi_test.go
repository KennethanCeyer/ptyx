package ptyx

import (
	"testing"
)

func TestCSI(t *testing.T) {
	tests := []struct {
		name string
		seq  string
		want string
	}{
		{"Simple sequence", "2J", "\x1b[2J"},
		{"Empty sequence", "", "\x1b["},
		{"Sequence with numbers and symbols", "1;31m", "\x1b[1;31m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CSI(tt.seq); got != tt.want {
				t.Errorf("CSI(%q) = %q; want %q", tt.seq, got, tt.want)
			}
		})
	}
}

func TestCUP(t *testing.T) {
	tests := []struct {
		name string
		row  int
		col  int
		want string
	}{
		{"Standard position", 10, 20, "\x1b[10;20H"},
		{"Top-left corner", 1, 1, "\x1b[1;1H"},
		{"Zero values", 0, 0, "\x1b[0;0H"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CUP(tt.row, tt.col); got != tt.want {
				t.Errorf("CUP(%d, %d) = %q; want %q", tt.row, tt.col, got, tt.want)
			}
		})
	}
}

func TestSGR(t *testing.T) {
	tests := []struct {
		name  string
		codes []int
		want  string
	}{
		{"Reset (no codes)", []int{}, "\x1b[0m"},
		{"Single code", []int{31}, "\x1b[31m"},
		{"Multiple codes", []int{1, 32, 44}, "\x1b[1;32;44m"},
		{"Zero code", []int{0}, "\x1b[0m"},
		{"Negative code", []int{-1}, "\x1b[-1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SGR(tt.codes...); got != tt.want {
				t.Errorf("SGR(%v) = %q; want %q", tt.codes, got, tt.want)
			}
		})
	}
}
