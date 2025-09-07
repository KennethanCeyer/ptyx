package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shell := "sh"
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
	}

	fmt.Printf("[DEMO] Spawning shell '%s' in a PTY...\n", shell)
	s, err := ptyx.Spawn(ctx, ptyx.SpawnOpts{Prog: shell})
	if err != nil {
		log.Fatalf("Failed to spawn: %v", err)
	}
	defer s.Close()

	if os.Getenv("PTYX_TEST_MODE") == "" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			fmt.Fprintln(os.Stderr, "\n[DEMO] Automatically cancelling sequence...")
			cancel()
		}()
	}

	fmt.Println("[DEMO] Running command sequence...")
	waitErr := runCommandSequence(ctx, s)

	fmt.Println("\n--- Wait Result ---")
	if waitErr != nil {
		fmt.Printf("Go error: %v\n", waitErr)
		var exitErr *ptyx.ExitError
		if errors.As(waitErr, &exitErr) {
			fmt.Printf("Exit code: %d\n", exitErr.ExitCode)
			if runtime.GOOS != "windows" {
				if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
					fmt.Printf("Terminated by signal: %s\n", ws.Signal())
				}
			}
		}
		if ctx.Err() != nil {
			fmt.Println("[DEMO] Process was interrupted.")
		}
		os.Exit(1)
	} else {
		fmt.Println("\n[DEMO] Process finished naturally.")
		fmt.Println("Process exited successfully with code 0.")
	}
}

func runCommandSequence(ctx context.Context, s ptyx.Session) error {
	const marker = "PTYX_CMD_DONE"
	cmdDone := make(chan struct{}, 1)

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := s.PtyReader().Read(buf)
			if n > 0 {
				if _, writeErr := os.Stdout.Write(buf[:n]); writeErr != nil {
					break
				}

				if strings.Contains(string(buf[:n]), marker) {
					select {
					case cmdDone <- struct{}{}:
					default:
					}
				}
			}
			if err != nil {
				break
			}
		}
	}()

	run := func(cmd string) error {
		separator := ";"
		if runtime.GOOS == "windows" {
			separator = "&"
		}

		fullCmd := fmt.Sprintf("%s %s echo %s", cmd, separator, marker)
		if _, err := fmt.Fprintf(s.PtyWriter(), "%s\r\n", fullCmd); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-cmdDone:
			return nil
		}
	}

	var initialCmds []string
	var commands []string
	var loadingCmd string
	if runtime.GOOS == "windows" {
		initialCmds = []string{"@echo off"}
		loadingCmd = "echo Loading... & ping -n 2 127.0.0.1 > nul"
		commands = []string{
			"cd",
			loadingCmd,
		}
	} else {
		initialCmds = []string{"stty -echo"}
		loadingCmd = "echo Loading...; sleep 1"
		commands = []string{
			"pwd",
			loadingCmd,
		}
	}

	sequence := append(initialCmds, commands...)
	for _, cmd := range sequence {
		if err := run(cmd); err != nil {
			return s.Wait()
		}
	}

	if _, err := fmt.Fprintf(s.PtyWriter(), "exit 0\r\n"); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "\n[DEMO] Command sequence finished.")
	return s.Wait()
}
