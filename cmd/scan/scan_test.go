package main

import (
	"bytes"
	"io"
	"testing"
	"time"
)

type mockReader struct {
	chunks [][]byte
	idx    int
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	if r.idx >= len(r.chunks) {
		return 0, io.EOF
	}
	n = copy(p, r.chunks[r.idx])
	r.idx++
	return n, nil
}

func TestPromptDetector(t *testing.T) {
	prompt := []byte("prompt: ")

	t.Run("PromptInSingleRead", func(t *testing.T) {
		input := bytes.NewReader([]byte("some data before prompt: and after"))
		promptFound := make(chan struct{})
		detector := &promptDetector{
			r:           input,
			prompt:      prompt,
			promptFound: promptFound,
		}
		go io.Copy(io.Discard, detector)

		select {
		case <-promptFound:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for prompt")
		}
	})

	t.Run("PromptSplitAcrossReads", func(t *testing.T) {
		mockIn := &mockReader{
			chunks: [][]byte{
				[]byte("some data before pro"),
				[]byte("mpt: and after"),
			},
		}
		promptFound := make(chan struct{})
		detector := &promptDetector{
			r:           mockIn,
			prompt:      prompt,
			promptFound: promptFound,
		}
		go io.Copy(io.Discard, detector)

		select {
		case <-promptFound:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for prompt")
		}
	})

	t.Run("NoPrompt", func(t *testing.T) {
		input := bytes.NewReader([]byte("some other data"))
		promptFound := make(chan struct{})
		detector := &promptDetector{r: input, prompt: prompt, promptFound: promptFound}
		go io.Copy(io.Discard, detector)

		select {
		case <-promptFound:
			t.Fatal("prompt was found but should not have been")
		case <-time.After(50 * time.Millisecond):
		}
	})
}
