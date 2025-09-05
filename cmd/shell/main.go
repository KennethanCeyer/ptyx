package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/KennethanCeyer/ptyx"
	"github.com/KennethanCeyer/ptyx/cmd/internal"
)

func main() {
	flag.Parse()

	opts := ptyx.SpawnOpts{
		Prog: defaultShell(),
		Args: flag.Args(),
	}
	err := internal.RunInPty(context.Background(), opts)
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
