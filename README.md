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

# Run a sequence of commands in a shell and handle interruption
go run ./cmd/sequence

# Run a shell inside a PTY, which itself runs inside a PTY
go run ./cmd/nested

# Resize bridge
go run ./cmd/resize -- /bin/sh

# Run an arbitrary command in a PTY
go run ./cmd/run -- bash -lc "echo hi; read -p 'press:' x; echo done"
```

## Use as a library

`ptyx` is designed to be simple to use. Here are a few examples showing how to accomplish common tasks.

### 1. Spawning a Non-Interactive Process

This is the most basic use case: running a command in a pseudo-terminal and streaming its output.

```go
package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	// Spawn a command in a new PTY session.
	// A context is used for cancellation.
	s, err := ptyx.Spawn(context.Background(), ptyx.SpawnOpts{
		Prog: "ping",
		Args: []string{"8.8.8.8"},
	})
	if err != nil {
		log.Fatalf("spawn failed: %v", err)
	}
	// Ensure the session is closed to clean up resources.
	defer s.Close()

	// Stream the PTY output to standard out.
	go io.Copy(os.Stdout, s.PtyReader())

	// Wait for the process to exit.
	if err := s.Wait(); err != nil {
		log.Printf("process wait failed: %v", err)
	}
}
```

### 2. Creating a Full Interactive Shell

For interactive applications like a terminal emulator, you need to connect the user's TTY to the PTY session. `ptyx` makes this easy.

The following example creates a complete, cross-platform interactive shell.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	// 1. Get a handle to the local console/TTY.
	c, err := ptyx.NewConsole()
	if err != nil {
		log.Fatalf("failed to create console: %v", err)
	}
	defer c.Close()

	// 2. Enable virtual terminal processing for color support (especially on Windows).
	c.EnableVT()

	// 3. Set the TTY to raw mode to pass all key presses directly to the PTY.
	st, err := c.MakeRaw()
	if err == nil {
		defer c.Restore(st)
	}

	// 4. Get the initial terminal size.
	w, h := c.Size()
	shell := "sh"
	if runtime.GOOS == "windows" {
		shell = "powershell.exe"
	}

	// 5. Spawn the shell in a new PTY with the correct dimensions.
	s, err := ptyx.Spawn(context.Background(), ptyx.SpawnOpts{Prog: shell, Cols: w, Rows: h})
	if err != nil {
		log.Fatalf("failed to spawn: %v", err)
	}
	defer s.Close()

	// 6. Create a multiplexer to bridge I/O between the local TTY and the PTY.
	m := ptyx.NewMux()
	if err := m.Start(c, s); err != nil {
		log.Fatalf("failed to start mux: %v", err)
	}
	defer m.Stop()

	// 7. Handle terminal resize events.
	go func() {
		for range c.OnResize() {
			_ = s.Resize(c.Size())
		}
	}()

	// 8. Wait for the PTY session to end.
	if err := s.Wait(); err != nil {
		if exitErr, ok := err.(*ptyx.ExitError); ok {
			fmt.Printf("\nProcess exited with code %d\n", exitErr.ExitCode)
		}
	}
}
```

### 3. Cancelling a Process

You can gracefully terminate a PTY session by cancelling its `context`. A common use case is handling user interruptions (e.g., `Ctrl+C`).

```go
package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	// Create a context that is cancelled when an interrupt signal is received.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	log.Println("Spawning 'ping' process. Press Ctrl+C to terminate.")
	s, err := ptyx.Spawn(ctx, ptyx.SpawnOpts{Prog: "ping", Args: []string{"8.8.8.8"}})
	if err != nil {
		log.Fatalf("spawn failed: %v", err)
	}
	defer s.Close()

	go io.Copy(os.Stdout, s.PtyReader())

	// Wait for the process to exit. This will block until the context is cancelled.
	if err := s.Wait(); err != nil {
		log.Printf("Process terminated: %v", err)
	}
}
```

### API References

```go
type Console interface {
  In() io.Reader
  Out() io.Writer
  Err() *os.File
  IsATTYOut() bool
  Size() (int, int)
  MakeRaw() (RawState, error)
  Restore(RawState) error
  EnableVT()
  OnResize() <-chan struct{}
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

type RawState interface{}
```

## Notes

- Unix/macOS/WSL: full PTY support using openpty or /dev/ptmx.
- Windows: Full ConPTY session support, console VT, and resize.
