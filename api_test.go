package ptyx

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsErrNotAConsole(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "is ErrNotAConsole",
			err:  ErrNotAConsole,
			want: true,
		},
		{
			name: "is a wrapped ErrNotAConsole",
			err:  fmt.Errorf("some context: %w", ErrNotAConsole),
			want: true,
		},
		{
			name: "is a different error",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "is nil",
			err:  nil,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsErrNotAConsole(tt.err); got != tt.want {
				t.Errorf("IsErrNotAConsole() = %v, want %v", got, tt.want)
			}
		})
	}
}
