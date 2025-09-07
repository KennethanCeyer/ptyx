package main

import (
	"flag"
	"fmt"

	"github.com/KennethanCeyer/ptyx"
)

type Options struct {
	Prog string
	Args []string
	Cols int
	Rows int
	Dir  string
	Env  []string
}

func ParseRunOpts(argv []string) (ptyx.SpawnOpts, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var cols, rows int
	var dir string
	var env multiString
	fs.IntVar(&cols, "cols", 0, "")
	fs.IntVar(&rows, "rows", 0, "")
	fs.StringVar(&dir, "dir", "", "")
	fs.Var(&env, "env", "KEY=VAL (repeatable)")
	fs.SetOutput(nil)

	if err := fs.Parse(argv); err != nil {
		return ptyx.SpawnOpts{}, err
	}
	opts := ptyx.SpawnOpts{
		Cols: cols,
		Rows: rows,
		Dir:  dir,
		Env:  env,
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

type multiString []string

func (m *multiString) String() string { return "" }
func (m *multiString) Set(s string) error {
	*m = append(*m, s)
	return nil
}
