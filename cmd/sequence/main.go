package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shell := "sh"
	if runtime.GOOS == "windows" {
		shell = "powershell.exe"
	}

	fmt.Printf("[DEMO] Spawning shell '%s' in a PTY...\n", shell)
	s, err := ptyx.Spawn(ctx, ptyx.SpawnOpts{Prog: shell})
	if err != nil {
		log.Fatalf("Failed to spawn: %v", err)
	}
	defer s.Close()

	readyCh := make(chan struct{}, 1)
	cmdCh := make(chan string)
	waitCh := make(chan error, 1)
	const commandDoneMarker = "COMMAND_DONE_MARKER_v1"

	go func() {
		var buf [4096]byte
		var readData bytes.Buffer
		for {
			n, err := s.PtyReader().Read(buf[:])
			if n > 0 {
				os.Stdout.Write(buf[:n])
				readData.Write(buf[:n])

				for bytes.Contains(readData.Bytes(), []byte(commandDoneMarker)) {
					idx := bytes.Index(readData.Bytes(), []byte(commandDoneMarker))

					select {
					case readyCh <- struct{}{}:
					default:
					}
					readData.Next(idx + len(commandDoneMarker))
				}
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		waitCh <- s.Wait()
	}()

	go func() {
		for cmd := range cmdCh {
			if _, err := fmt.Fprintf(s.PtyWriter(), "%s\r\n", cmd); err != nil {
				log.Printf("Failed to write to PTY: %v", err)
				return
			}
		}
		fmt.Fprintln(os.Stderr, "\n[DEMO] Command channel closed.")
	}()

	if os.Getenv("PTYX_TEST_MODE") == "" {
		go func() {
			time.Sleep(150 * time.Millisecond)
			fmt.Fprintln(os.Stderr, "\n[DEMO] Automatically cancelling sequence...")
			cancel()
		}()
	}

	go runCommandSequence(ctx, cmdCh, readyCh, commandDoneMarker)

	var waitErr error
	fmt.Println("[DEMO] Running command sequence...")
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\n[DEMO] Cancellation detected. Terminating process...")
		waitErr = <-waitCh

	case waitErr = <-waitCh:
		fmt.Println("\n[DEMO] Process finished naturally.")
	}

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
	} else {
		fmt.Println("Process exited successfully with code 0.")
	}
}

func runCommandSequence(ctx context.Context, cmdCh chan<- string, readyCh <-chan struct{}, marker string) {
	defer close(cmdCh)

	sendCommand := func(cmd string) bool {
		fmt.Fprintf(os.Stderr, "\n[DEMO] Sending command: %s\n", cmd)
		fullCmd := fmt.Sprintf("%s; echo %s", cmd, marker)
		if cmd == "" {
			fullCmd = fmt.Sprintf("echo %s", marker)
		}

		select {
		case cmdCh <- fullCmd:
			return true
		case <-ctx.Done():
			return false
		}
	}

	waitReady := func() bool {
		select {
		case <-readyCh:
			return true
		case <-ctx.Done():
			return false
		}
	}

	var loadingCmd string
	if runtime.GOOS == "windows" {
		loadingCmd = "Write-Host 'Loading...'; for ($i=0; $i -le 25; $i++) { $p = '#' * $i; $r = ' ' * (25 - $i); $pct = $i * 4; Write-Host -NoNewline \"[$p$r] - $pct%`r\"; Start-Sleep -Milliseconds 100 }; echo ''"
	} else {
		loadingCmd = "echo 'Loading...'; i=0; while [ $i -le 25 ]; do printf '['; j=0; while [ $j -lt $i ]; do printf '#'; j=$((j+1)); done; j=0; while [ $j -lt $((25-i)) ]; do printf ' '; j=$((j+1)); done; printf '] - %s%%' $((i*4)); printf '\r'; sleep 0.1; i=$((i+1)); done; echo ''"
	}

	commands := []string{
		"echo '--- Starting command sequence ---'",
		"echo 'Current directory:'; pwd",
		loadingCmd,
		"echo '--- Sequence finished ---'",
		"exit 0",
	}

	lastCmd := commands[len(commands)-1]
	commands = commands[:len(commands)-1]

	if runtime.GOOS != "windows" {
		commands = append([]string{"stty -echo"}, commands...)
	} else {
		commands = append([]string{
			"Set-PSReadlineOption -HistorySaveStyle SaveNothing",
			"Remove-Module PSReadline",
		}, commands...)
	}

	if !sendCommand(commands[0]) {
		return
	}

	for _, cmd := range commands[1:] {
		if !waitReady() {
			return
		}
		if !sendCommand(cmd) {
			return
		}
	}

	if !waitReady() {
		return
	}

	select {
	case cmdCh <- lastCmd:
	case <-ctx.Done():
	}
}
