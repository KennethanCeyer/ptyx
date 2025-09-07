package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestColorMain(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	main()

	w.Close()
	os.Stdout = oldStdout
	<-outC
}
