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

func parseResizeOpts(args []string) (*ptyx.SpawnOpts, error) {
	if len(args) == 0 {
		return nil, errors.New("usage: resize -- <prog> [args...]")
	}
	return &ptyx.SpawnOpts{
		Prog: args[0],
		Args: args[1:],
	}, nil
}

func main() {
	flag.Parse()
	opts, err := parseResizeOpts(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := internal.RunInPty(context.Background(), *opts); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
