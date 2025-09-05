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

func getProjectRoot() (string, error) {
	_, b, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("cannot determine project root: runtime.Caller failed")
	}
	projectRoot := filepath.Join(filepath.Dir(b), "..", "..")
	return projectRoot, nil
}

func main() {
	projectRoot, err := getProjectRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("--- Running 'go run ./cmd/shell' in a PTY (nested PTY) ---")

	opts := ptyx.SpawnOpts{
		Prog: "go",
		Args: []string{"run", "./cmd/shell"},
		Dir:  projectRoot,
	}

	err = internal.RunInPty(context.Background(), opts)
	fmt.Println("\n--- Nested shell test finished ---")
	if err != nil {
		if _, ok := err.(*ptyx.ExitError); !ok {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
	}
}
