package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestScanTargetMain(t *testing.T) {
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	rIn, wIn, _ := os.Pipe()
	os.Stdin = rIn

	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	input := "PTYX\n"
	go func() {
		defer wIn.Close()
		_, _ = wIn.Write([]byte(input))
	}()

	main()
	_ = wOut.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)

	output := buf.String()
	expected := "What is your name? Hello, PTYX! Welcome to ptyx.\n"
	if output != expected {
		t.Errorf("main() output = %q, want %q", output, expected)
	}
}
