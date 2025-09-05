package ptyx

import (
	"io"
	"os"
	"sync"
)

type console struct {
	in, out, err *os.File
	outTTY, errTTY bool
	win *winWatcher
	raw RawState
	closeOnce sync.Once
}

func (c *console) In() io.Reader  { return c.in }
func (c *console) Out() io.Writer { return c.out }
func (c *console) Err() *os.File { return c.err }
func (c *console) IsATTYOut() bool { return c.outTTY }
func (c *console) IsATTYErr() bool { return c.errTTY }
func (c *console) OnResize() <-chan struct{} { return c.win.C }
