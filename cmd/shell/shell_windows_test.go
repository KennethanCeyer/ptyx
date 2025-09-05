//go:build windows

package main

import "testing"

func TestDefaultShell_Windows(t *testing.T) {
	if got := defaultShell(); got != "cmd.exe" {
		t.Errorf("defaultShell() = %q, want %q", got, "cmd.exe")
	}
}
