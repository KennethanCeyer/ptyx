package main

import (
	"flag"
	"fmt"

	"github.com/KennethanCeyer/ptyx"
)

type Options struct {
	Cols int
	Rows int
	Pid  int
}

func ParseResizeOpts(argv []string) (ptyx.SpawnOpts, error) {
	fs := flag.NewFlagSet("resize", flag.ContinueOnError)
	var cols, rows int
	fs.IntVar(&cols, "cols", 0, "")
	fs.IntVar(&rows, "rows", 0, "")
	fs.SetOutput(nil)

	if err := fs.Parse(argv); err != nil {
		return ptyx.SpawnOpts{}, err
	}
	opts := ptyx.SpawnOpts{
		Cols: cols,
		Rows: rows,
	}
	args := fs.Args()
	if len(args) > 0 {
		opts.Prog = args[0]
		opts.Args = args[1:]
	} else {
		return ptyx.SpawnOpts{}, fmt.Errorf("no program specified")
	}
	return opts, nil
}
