package ptyx

import (
	"fmt"
	"testing"
)

func TestCSI(t *testing.T) {
	seq := "2J"
	expected := "\x1b[2J"
	if got := CSI(seq); got != expected {
		t.Errorf("CSI(%q) = %q; want %q", seq, got, expected)
	}
}

func TestCUP(t *testing.T) {
	row, col := 10, 20
	expected := fmt.Sprintf("\x1b[%d;%dH", row, col)
	if got := CUP(row, col); got != expected {
		t.Errorf("CUP(%d, %d) = %q; want %q", row, col, got, expected)
	}
}

func TestSGR(t *testing.T) {
	tests := []struct {
		name  string
		codes []int
		want  string
	}{
		{"Reset", []int{}, "\x1b[0m"},
		{"SingleCode", []int{31}, "\x1b[31m"},
		{"MultipleCodes", []int{1, 32, 44}, "\x1b[1;32;44m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SGR(tt.codes...); got != tt.want {
				t.Errorf("SGR(%v) = %q; want %q", tt.codes, got, tt.want)
			}
		})
	}
}
