package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/KennethanCeyer/ptyx/cmd/internal"
)

func main() {
	flag.Parse()
	opts, err := ParseRunOpts(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if err := internal.RunInPty(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
