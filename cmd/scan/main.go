package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/KennethanCeyer/ptyx"
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
	_, b, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Error: cannot determine project root")
	}
	projectRoot := filepath.Join(filepath.Dir(b), "..", "..")

	targetProg := "go"
	targetArgs := []string{"run", "./cmd/internal/scan-target"}

	fmt.Println("--- Spawning a program that waits for `Scanln` in a PTY. ---")

	s, err := ptyx.Spawn(ptyx.SpawnOpts{
		Prog: targetProg,
		Args: targetArgs,
		Dir:  projectRoot,
	})
	if err != nil {
		log.Fatalf("Failed to spawn: %v", err)
	}
	defer s.Close()

	promptFound := make(chan struct{})
	detector := &promptDetector{
		r:           s.PtyReader(),
		prompt:      []byte("What is your name? "),
		promptFound: promptFound,
	}

	go io.Copy(os.Stdout, detector)

	select {
	case <-promptFound:
	case <-time.After(10 * time.Second):
		log.Fatal("Timeout: Did not find expected prompt in PTY output.")
	}

	inputToSend := "World"
	fmt.Fprintf(os.Stdout, "\n\n[DEMO] Found prompt. Sending input '%s\\n' to the PTY to unblock Scanln...\n", inputToSend)

	_, err = fmt.Fprintf(s.PtyWriter(), "%s\r\n", inputToSend)
	if err != nil {
		log.Fatalf("Failed to write to PTY: %v", err)
	}

	s.Wait()

	fmt.Println("\n[DEMO] Program finished.")
}
