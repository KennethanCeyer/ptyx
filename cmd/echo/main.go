package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/KennethanCeyer/ptyx"
)

var (
	newConsoleFunc = ptyx.NewConsole
)

func echoLoop(in io.Reader, out io.Writer, errOut io.Writer) {
	buf := make([]byte, 1024)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			data := buf[:n]
			if stopIdx := bytes.IndexByte(data, 3); stopIdx != -1 {
				if stopIdx > 0 {
					fmt.Fprintf(out, "read %d bytes: %q\r\n", stopIdx, data[:stopIdx])
				}
				break
			}
			fmt.Fprintf(out, "read %d bytes: %q\r\n", n, data)
		}
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(errOut, "read error: %v\r\n", err)
			}
			break
		}
	}
}

func main() {
	c, err := newConsoleFunc()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create console: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	st, err := c.MakeRaw()
	if err != nil {
		fmt.Fprintln(c.Err(), "raw mode error:", err)
		return
	}
	defer c.Restore(st)

	fmt.Fprint(c.Out(), "Entering raw echo mode. Press Ctrl+C to exit.\r\n")
	echoLoop(c.In(), c.Out(), c.Err())
}
