package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	baseCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	shell := "sh"
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
	}

	spawnCtx, cancel := context.WithTimeout(baseCtx, 15*time.Second)
	defer cancel()

	fmt.Printf("[DEMO] Spawning shell '%s' in a PTY...\n", shell)
	s, err := ptyx.Spawn(spawnCtx, ptyx.SpawnOpts{Prog: shell})
	if err != nil { log.Fatalf("Failed to spawn: %v", err) }
	defer s.Close()

	if os.Getenv("PTYX_TEST_MODE") == "" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			fmt.Fprintln(os.Stderr, "\n[DEMO] Automatically cancelling sequence...")
			cancel()
		}()
	}

	fmt.Println("[DEMO] Running command sequence...")
	waitErr := runCommandSequence(s)

	exitCode, output := handleWaitResult(spawnCtx, waitErr)
	fmt.Print(output)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
func runCommandSequence(s ptyx.Session) error {
	const ready = "[[PTYX_READY]]"
	const done = "[[PTYX_DONE]]"

	cmdDone := make(chan struct{}, 1)

	go func() {
		sc := bufio.NewScanner(s.PtyReader())
		buf := make([]byte, 0, 1<<20)
		sc.Buffer(buf, 1<<20)
		for sc.Scan() {
			line := strings.TrimRight(sc.Text(), "\r")
			fmt.Fprintln(os.Stdout, line)
			if strings.TrimSpace(line) == done {
				select { case cmdDone <- struct{}{}: default: }
			}
		}
	}()

	var full string
	if runtime.GOOS == "windows" {
		full = `@echo [[PTYX_READY]] & @echo Loading... & ping -n 3 127.0.0.1 >NUL & @echo [[PTYX_DONE]] & exit 0`
	} else {
		full = `echo [[PTYX_READY]]; echo Loading...; sleep 2; echo [[PTYX_DONE]]; exit 0`
	}

	if _, err := fmt.Fprintf(s.PtyWriter(), "%s\r\n", full); err != nil {
		return err
	}

	return s.Wait()
}

func handleWaitResult(spawnCtx context.Context, waitErr error) (int, string) {
	var b strings.Builder
	b.WriteString("\n--- Wait Result ---\n")
	if waitErr != nil {
		fmt.Fprintf(&b, "Go error: %v\n", waitErr)
		var exitErr *ptyx.ExitError
		if errors.As(waitErr, &exitErr) {
			fmt.Fprintf(&b, "Exit code: %d\n", exitErr.ExitCode)
			b.WriteString(checkSignal(waitErr))
		}
		if spawnCtx.Err() != nil {
			b.WriteString("[DEMO] Process was interrupted.\n")
		}
		return 1, b.String()
	}

	b.WriteString("\n[DEMO] Process finished naturally.\n")
	b.WriteString("Process exited successfully with code 0.\n")
	return 0, b.String()
}
