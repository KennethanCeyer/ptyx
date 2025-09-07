package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

var (
	runtimeCallerFunc = runtime.Caller
	ptyxSpawnFunc     = ptyx.Spawn
	timeAfterFunc     = time.After
	stdout            = io.Writer(os.Stdout)
)

type promptDetector struct {
	r           io.Reader
	prompt      []byte
	promptFound chan struct{}
	once        sync.Once
	mu          sync.Mutex
	readBuf     bytes.Buffer
}

func (d *promptDetector) Read(p []byte) (n int, err error) {
	n, err = d.r.Read(p)
	if n > 0 {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.readBuf.Write(p[:n])
		if bytes.Contains(d.readBuf.Bytes(), d.prompt) {
			d.once.Do(func() { close(d.promptFound) })
		}
	}
	return
}

func main() {
	_, b, _, ok := runtimeCallerFunc(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: cannot determine project root")
		os.Exit(1)
	}
	projectRoot := filepath.Join(filepath.Dir(b), "..", "..")

	targetProg := "go"
	targetArgs := []string{"run", "./cmd/internal/scan-target"}

	fmt.Fprintln(stdout, "--- Spawning a program that waits for `Scanln` in a PTY. ---")

	s, err := ptyxSpawnFunc(context.Background(), ptyx.SpawnOpts{
		Prog: targetProg,
		Args: targetArgs,
		Dir:  projectRoot,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to spawn: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	promptFound := make(chan struct{})
	detector := &promptDetector{
		r:           s.PtyReader(),
		prompt:      []byte("What is your name? "),
		promptFound: promptFound,
	}

	go io.Copy(stdout, detector)

	select {
	case <-promptFound:
	case <-timeAfterFunc(10 * time.Second):
		fmt.Fprintln(os.Stderr, "Timeout: Did not find expected prompt in PTY output.")
		os.Exit(1)
	}

	inputToSend := "World"
	fmt.Fprintf(stdout, "\n\n[DEMO] Found prompt. Sending input '%s\\n' to the PTY to unblock Scanln...\n", inputToSend)

	_, err = fmt.Fprintf(s.PtyWriter(), "%s\r\n", inputToSend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write to PTY: %v\n", err)
		os.Exit(1)
	}

	s.Wait()

	fmt.Fprintln(stdout, "\n[DEMO] Program finished.")
}
