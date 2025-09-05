package ptyx

import (
	"testing"
)

func TestExitError_Error(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		want     string
	}{
		{"Non-zero exit code", 127, "process exited with status 127"},
		{"Zero exit code", 0, "process exited with status 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ExitError{ExitCode: tt.exitCode}
			if got := e.Error(); got != tt.want {
				t.Errorf("ExitError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
