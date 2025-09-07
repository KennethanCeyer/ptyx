package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

func TestSequenceHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_SEQUENCE") == "1" {
		main()
		return
	}
}

func TestSequence_NormalCompletion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := ptyx.SpawnOpts{
		Prog: os.Args[0],
		Args: []string{"-test.run=^TestSequenceHelperProcess$"},
		Env: append(os.Environ(),
			"GO_TEST_SEQUENCE=1",
			"PTYX_TEST_MODE=1",
		),
	}
	s, err := ptyx.Spawn(ctx, opts)
	if err != nil {
		t.Fatalf("failed to spawn command in pty: %v", err)
	}
	defer s.Close()

	var out bytes.Buffer
	readerDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&out, s.PtyReader())
		close(readerDone)
	}()

	waitDone := make(chan error, 1)
	go func() { waitDone <- s.Wait() }()

	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("command failed with error: %v\nOutput:\n%s", err, out.String())
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for normal completion; output:\n%s", out.String())
	}

	<-readerDone
	output := out.String()

	expected := []string{
		"[[PTYX_READY]]",
		"Loading...",
		"Process finished naturally.",
		"Process exited successfully with code 0.",
	}
	for _, sub := range expected {
		if !strings.Contains(output, sub) {
			t.Errorf("expected output to contain %q, but it didn't.\nFull output:\n%s", sub, output)
		}
	}
}

func TestSequence_Interruption(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := ptyx.SpawnOpts{
		Prog: os.Args[0],
		Args: []string{"-test.run=^TestSequenceHelperProcess$"},
		Env: append(os.Environ(),
			"GO_TEST_SEQUENCE=1",
			"PTYX_TEST_MODE=1",
		),
	}
	s, err := ptyx.Spawn(ctx, opts)
	if err != nil {
		t.Fatalf("failed to spawn command in pty: %v", err)
	}
	defer s.Close()

	ready := make(chan struct{})
	var out bytes.Buffer

	go func() {
		sc := bufio.NewScanner(io.TeeReader(s.PtyReader(), &out))
		sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
		for sc.Scan() {
			line := strings.TrimRight(sc.Text(), "\r")
			trim := strings.TrimSpace(line)

			if strings.Contains(trim, "[[PTYX_READY]]") || strings.Contains(trim, "Loading...") {
				select { case <-ready: default: close(ready) }
				cancel()
				return
			}
		}
	}()

	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for READY/Loading..., output:\n%s", out.String())
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- s.Wait() }()

	select {
	case waitErr := <-waitDone:
		if waitErr == nil {
			t.Fatalf("expected command to fail due to interruption, but it succeeded.\nOutput:\n%s", out.String())
		}
	case <-ctx.Done():
		waitErr := <-waitDone
		if waitErr == nil {
			t.Fatalf("expected failure after cancel, but got nil")
		}
	}

	if strings.Contains(out.String(), "[DEMO] Process finished naturally.") {
		t.Errorf("sequence should have been interrupted before finishing.\nOutput:\n%s", out.String())
	}
}
