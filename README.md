<h1 align="center">ptyx — Cross-Platform PTY/TTY Toolkit</h1>

<p align="center">
  <img width="250" src="docs/logo.svg" alt="ptyx logo" />
<p>

<p align="center">
  <a href="https://github.com/KennethanCeyer/ptyx/actions/workflows/ci.yaml"><img alt="CI Status" src="https://github.com/KennethanCeyer/ptyx/actions/workflows/ci.yaml/badge.svg"></a>
  <a href="https://go.dev"><img alt="Go" src="https://img.shields.io/badge/Go-%3E=1.24-00ADD8?logo=go"></a>
  <a href="https://pkg.go.dev/github.com/KennethanCeyer/ptyx"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/KennethanCeyer/ptyx.svg"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-yellow.svg"></a>
  <img alt="Platform" src="https://img.shields.io/badge/Platform-macOS%20|%20Linux%20|%20Windows-blue.svg">
</p>

---

## Overview

`ptyx` is a Go library that provides a simple, cross-platform API for managing pseudo-terminals (PTY) and terminal TTYs.

## Features

- **Cross-Platform PTY**: Simple API to spawn processes in a pseudo-terminal on macOS, Linux, BSDs (using `ptmx`) and on Windows (using `ConPTY`).
- **TTY Control**: Functions to control the local terminal, including setting raw mode, getting terminal size, and receiving resize notifications.
- **I/O Bridge**: A `Mux` utility to easily connect the local terminal's stdin/stdout to the PTY session.
- **Zero External Dependencies**: Relies only on the standard library and the official `golang.org/x` packages (`sys`, `term`).

---

## Installation

```bash
go get github.com/KennethanCeyer/ptyx
```

## Run the demos

```bash
# Interactive shell
go run ./cmd/shell

# Local spinner/progress (no PTY)
go run ./cmd/spinner

# Show terminal color support
go run ./cmd/color

# Test ANSI color passthrough by running the color demo in a PTY
go run ./cmd/passthrough

# Raw stdin echo
go run ./cmd/echo

# Capture and parse terminal output as events
go run ./cmd/event

# Send input to a program waiting in a PTY
go run ./cmd/scan

# Run a shell inside a PTY, which itself runs inside a PTY
go run ./cmd/nested

# Resize bridge
go run ./cmd/resize -- /bin/sh

# Run an arbitrary command in a PTY
go run ./cmd/run -- bash -lc "echo hi; read -p 'press:' x; echo done"
```

## Use as a library

```go
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	c, err := ptyx.NewConsole()
	if err != nil {
		log.Fatalf("failed to create console: %v", err)
	}
	defer c.Close()
	c.EnableVT()

	st, err := c.MakeRaw()
	if err != nil {
		log.Printf("failed to enter raw mode (this is expected when not in a TTY): %v", err)
	} else {
		defer c.Restore(st)
	}

	w, h := c.Size()
	shell := "sh"
	if runtime.GOOS == "windows" {
		shell = "powershell.exe"
	}
	s, err := ptyx.Spawn(ptyx.SpawnOpts{Prog: shell, Cols: w, Rows: h})
	if err != nil {
		log.Fatalf("failed to spawn: %v", err)
	}
	defer s.Close()

	m := ptyx.NewMux()
	if err := m.Start(c, s); err != nil {
		log.Fatalf("failed to start mux: %v", err)
	}
	defer m.Stop()

	go func() {
		for range c.OnResize() {
			if err := s.Resize(c.Size()); err != nil {
				log.Printf("failed to resize: %v", err)
			}
		}
	}()

	if err := s.Wait(); err != nil {
		if exitErr, ok := err.(*ptyx.ExitError); ok {
			fmt.Printf("\nProcess exited with code %d\n", exitErr.ExitCode)
		} else {
			log.Fatalf("\nwait failed: %v", err)
		}
	}
}
```

### Run a program in PTY

```go
s, err := ptyx.Spawn(ptyx.SpawnOpts{Prog: "ping", Args: []string{"8.8.8.8"}})
if err != nil {
	log.Fatalf("spawn failed: %v", err)
}
defer s.Close()

go io.Copy(os.Stdout, s.PtyReader())
go io.Copy(s.PtyWriter(), os.Stdin)

if err := s.Wait(); err != nil {
	log.Printf("process wait failed: %v", err)
}
```

### API References

```go
type Console interface {
  Size() (int, int)
  MakeRaw() (RawState, error)
  Restore(RawState) error
  EnableVT()
  OnResize() <-chan struct{}
  In() io.Reader
  Out() io.Writer
  Err() io.Writer
  Close() error
}

type Session interface {
  PtyReader() io.Reader
  PtyWriter() io.Writer
  Resize(cols, rows int) error
  Wait() error
  Kill() error
  Close() error
  Pid() int
  CloseStdin() error
}

type Mux interface {
  Start(c Console, s Session) error
  Stop() error
}

type SpawnOpts struct {
  Prog string
  Args []string
  Env  []string
  Dir  string
  Cols int
  Rows int
}

type ExitError struct {
  ExitCode int
}

// RawState is an opaque type that represents the terminal's state,
// returned by MakeRaw() and passed to Restore().
type RawState interface{}
```

## Notes

- Unix/macOS/WSL: full PTY support using openpty or /dev/ptmx.
- Windows: Full ConPTY session support, console VT, and resize.
