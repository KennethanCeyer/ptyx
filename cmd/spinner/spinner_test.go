package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/KennethanCeyer/ptyx"
	"github.com/KennethanCeyer/ptyx/testptyx"
)

func TestMain_NoConsole(t *testing.T) {
	c, err := ptyx.NewConsole()
	if c != nil || !errors.Is(err, ptyx.ErrNotAConsole) {
		t.Skip("test environment seems to have a console, skipping")
	}
	main()
}

func TestSpinnerMain(t *testing.T) {
	if os.Getenv("PTYX_TEST_RUN_MAIN") == "1" {
		main()
		return
	}

	t.Run("main execution", func(t *testing.T) {
		opts := ptyx.SpawnOpts{
			Prog: os.Args[0],
			Args: []string{"-test.run=^TestSpinnerMain$"},
			Env:  append(os.Environ(), "PTYX_TEST_RUN_MAIN=1"),
		}

		err := ptyx.Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("ptyx.Run() failed with %v", err)
		}
	})
}

func TestRunSpinner(t *testing.T) {
	t.Run("Normal completion", func(t *testing.T) {
		pr, _ := io.Pipe()
		defer pr.Close()
		mockConsole := &testptyx.MockConsole{
			InReader:  pr,
			OutBuffer: &bytes.Buffer{},
		}
		ctx := context.Background()

		runSpinner(ctx, mockConsole)

		output := mockConsole.OutBuffer.String()
		if !strings.Contains(output, "Done.") {
			t.Errorf("Expected output to contain 'Done.', but got: %q", output)
		}
		if !strings.Contains(output, ptyx.CSI("?25l")) {
			t.Errorf("Expected output to hide cursor, but it didn't.")
		}
		if !strings.Contains(output, ptyx.CSI("?25h")) {
			t.Errorf("Expected output to show cursor at the end, but it didn't.")
		}
	})

	t.Run("Context canceled", func(t *testing.T) {
		mockConsole := testptyx.NewMockConsole("")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		runSpinner(ctx, mockConsole)

		output := mockConsole.OutBuffer.String()
		if !strings.Contains(output, "Spinner stopped.") {
			t.Errorf("Expected output to contain 'Spinner stopped.', but got: %q", output)
		}
	})
}
