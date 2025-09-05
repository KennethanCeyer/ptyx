//go:build unix || darwin || linux || freebsd || netbsd || openbsd

package main

import "os"

func defaultShell() string {
	if sh := os.Getenv("SHELL"); sh != "" { return sh }
	return "/bin/sh"
}
