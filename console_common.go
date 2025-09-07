package ptyx

import (
	"errors"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

var ErrNotAConsole = errors.New("ptyx: not a console")

type resizeWatcher struct {
	C     chan struct{}
	stop  chan struct{}
	ready chan struct{}
}

type rawState struct{ st *term.State; fd int }

type console struct {
	in, out, err *os.File
	outTTY, errTTY bool
	raw            RawState
	win            *resizeWatcher
	closeOnce      sync.Once
}

func NewConsole() (Console, error) {
	c := &console{in: os.Stdin, out: os.Stdout, err: os.Stderr}
	if c.out == nil {
		return nil, ErrNotAConsole
	}
	if !term.IsTerminal(int(c.out.Fd())) { return nil, ErrNotAConsole }
	c.outTTY = true
	c.errTTY = term.IsTerminal(int(c.err.Fd()))

	c.initResizeWatcher()
	c.EnableVT()
	return c, nil
}

func (c *console) In() io.Reader {
	if c.in == nil {
		return nil
	}
	return c.in
}

func (c *console) Out() io.Writer {
	if c.out == nil {
		return nil
	}
	return c.out
}

func (c *console) Err() *os.File {
	if c.err == nil {
		return nil
	}
	return c.err
}

func (c *console) OnResize() <-chan struct{} {
	if c.win == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return c.win.C
}

func (c *console) Close() error {
	c.closeOnce.Do(func() {
		if c.win != nil && c.win.stop != nil {
			close(c.win.stop)
		}
	})
	return nil
}

func (c *console) IsATTYOut() bool {
	return c.outTTY
}

func (c *console) IsATTYErr() bool {
	return c.errTTY
}

func (c *console) Size() (int, int) {
	if c.out == nil {
		return 0, 0
	}
	w, h, err := term.GetSize(int(c.out.Fd()))
	if err != nil {
		return 0, 0
	}
	return w, h
}

func (c *console) MakeRaw() (RawState, error) {
	if c.in == nil {
		return nil, ErrNotAConsole
	}
	fd := int(c.in.Fd())
	if !term.IsTerminal(fd) {
		return nil, ErrNotAConsole
	}
	st, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	r := &rawState{st: st, fd: fd}
	c.raw = r
	return r, nil
}

func (c *console) Restore(s RawState) error {
	r, ok := s.(*rawState)
	if !ok || r == nil || r.st == nil {
		return nil
	}
	err := term.Restore(r.fd, r.st)
	if err == nil {
		c.raw = nil
	}
	return err
}
