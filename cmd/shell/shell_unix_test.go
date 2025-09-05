//go:build unix || darwin || linux || freebsd || netbsd || openbsd

package main

import (
	"testing"
)

func TestDefaultShell(t *testing.T) {
	t.Run("SHELL is set", func(t *testing.T) {
		t.Setenv("SHELL", "/bin/zsh")
		if got := defaultShell(); got != "/bin/zsh" {
			t.Errorf("defaultShell() = %q, want %q", got, "/bin/zsh")
		}
	})

	t.Run("SHELL is empty", func(t *testing.T) {
		t.Setenv("SHELL", "")
		if got := defaultShell(); got != "/bin/sh" {
			t.Errorf("defaultShell() = %q, want %q", got, "/bin/sh")
		}
	})
}
