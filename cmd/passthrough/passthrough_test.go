package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPassthroughMain(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	main()

	w.Close()
	var errBuf bytes.Buffer
	_, _ = io.Copy(&errBuf, r)
	os.Stderr = oldStderr

	if errBuf.Len() > 0 {
		if !strings.Contains(errBuf.String(), "The buffer is too small") {
			t.Errorf("main() produced unexpected output on stderr: %s", errBuf.String())
		}
	}
}
