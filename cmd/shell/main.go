package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/KennethanCeyer/ptyx"
)

var (
	runInteractiveFunc = ptyx.RunInteractive
)

func main() {
	flag.Parse()

	opts := ptyx.SpawnOpts{
		Prog: defaultShell(),
		Args: flag.Args(),
	}
	err := runInteractiveFunc(context.Background(), opts)
	if err == nil {
		return
	}

	var exitErr *ptyx.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode)
	} else {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
