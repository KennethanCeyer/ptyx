package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/KennethanCeyer/ptyx"
	"github.com/KennethanCeyer/ptyx/cmd/internal"
)

func main() {
	_, b, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: cannot determine project root")
		os.Exit(1)
	}
	projectRoot := filepath.Join(filepath.Dir(b), "..", "..")

	fmt.Println("--- Running 'go run ./cmd/color' in a PTY to test passthrough ---")

	opts := ptyx.SpawnOpts{
		Prog: "go",
		Args: []string{"run", "./cmd/color"},
		Dir:  projectRoot,
	}

	err := internal.RunInPty(context.Background(), opts)
	if err != nil {
		var exitErr *ptyx.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode != 0 {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	}
}
