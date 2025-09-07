package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/KennethanCeyer/ptyx"
)

var (
	getProjectRootFunc = getProjectRoot
	runInteractiveFunc = ptyx.RunInteractive
	runtimeCaller      = runtime.Caller
)

func getProjectRoot() (string, error) {
	_, b, _, ok := runtimeCaller(0)
	if !ok {
		return "", errors.New("cannot determine project root: runtime.Caller failed")
	}
	projectRoot := filepath.Join(filepath.Dir(b), "..", "..")
	return projectRoot, nil
}

func main() {
	projectRoot, err := getProjectRootFunc()
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

	err = runInteractiveFunc(context.Background(), opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
}
