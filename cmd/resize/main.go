package main

import (
	"context"
	"fmt"
	"os"

	"github.com/KennethanCeyer/ptyx"
)

var (
	parseResizeOptsFunc = ParseResizeOpts
	runInteractiveFunc  = ptyx.RunInteractive
)

func main() {
	opts, err := parseResizeOptsFunc(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := runInteractiveFunc(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
