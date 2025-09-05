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

func parseRunOpts(args []string) (*ptyx.SpawnOpts, error) {
	if len(args) == 0 {
		return nil, errors.New("usage: run -- <prog> [args...]")
	}
	return &ptyx.SpawnOpts{
		Prog: args[0],
		Args: args[1:],
	}, nil
}

func main() {
	flag.Parse()
	opts, err := parseRunOpts(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if err := internal.RunInPty(context.Background(), *opts); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
