package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/KennethanCeyer/ptyx"
)

type EventType string

const (
	EventText      EventType = "TEXT"
	EventANSI      EventType = "ANSI"
	EventControl   EventType = "CONTROL"
	EventUnhandled EventType = "UNHANDLED"
)

type Event struct {
	Type    EventType
	Payload string
}

func (e Event) String() string {
	return fmt.Sprintf("[EVENT:%s] %q", e.Type, e.Payload)
}

func main() {
	_, b, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: cannot determine project root")
		os.Exit(1)
	}
	projectRoot := filepath.Join(filepath.Dir(b), "..", "..")

	fmt.Println("--- Starting 'go run ./cmd/spinner' and capturing output as events ---")

	c, err := ptyx.NewConsole()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating console:", err)
		os.Exit(1)
	}
	defer c.Close()
	w, h := c.Size()

	opts := ptyx.SpawnOpts{
		Prog: "go",
		Args: []string{"run", "./cmd/spinner"},
		Dir:  projectRoot,
		Cols: w,
		Rows: h,
	}

	s, err := ptyx.Spawn(context.Background(), opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error spawning process:", err)
		os.Exit(1)
	}
	defer s.Close()

	go processStream(os.Stdout, s.PtyReader())

	if err := s.Wait(); err != nil && !errors.As(err, new(*ptyx.ExitError)) {
		fmt.Fprintln(os.Stderr, "Wait error:", err)
	}

	fmt.Println("\n--- Event stream terminated ---")
}

func processStream(w io.Writer, r io.Reader) {
	reader := bufio.NewReader(r)
	var textBuffer bytes.Buffer

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF && textBuffer.Len() > 0 {
				fmt.Fprintln(w, Event{Type: EventText, Payload: textBuffer.String()})
			}
			break
		}

		switch r {
		case '\x1b':
			if textBuffer.Len() > 0 {
				fmt.Fprintln(w, Event{Type: EventText, Payload: textBuffer.String()})
				textBuffer.Reset()
			}

			next, _, err := reader.ReadRune()
			if err != nil {
				break
			}
			if next == '[' {
				var csiSequence bytes.Buffer
				for {
					seqRune, _, seqErr := reader.ReadRune()
					if seqErr != nil {
						break
					}
					csiSequence.WriteRune(seqRune)
					if seqRune >= 0x40 && seqRune <= 0x7E {
						break
					}
				}
				fmt.Fprintln(w, Event{Type: EventANSI, Payload: csiSequence.String()})
			} else {
				fmt.Fprintln(w, Event{Type: EventUnhandled, Payload: string(next)})
			}
		case '\r', '\n':
			if textBuffer.Len() > 0 {
				fmt.Fprintln(w, Event{Type: EventText, Payload: textBuffer.String()})
				textBuffer.Reset()
			}
			fmt.Fprintln(w, Event{Type: EventControl, Payload: string(r)})
		default:
			textBuffer.WriteRune(r)
		}
	}
}
