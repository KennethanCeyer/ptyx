//go:build unix || darwin || linux || freebsd || netbsd || openbsd

package ptyx

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

type rawState struct{ st *term.State; fd int }
type winWatcher struct{ C chan struct{}; ch chan os.Signal }

func NewConsole() (Console, error) {
	c := &console{ in: os.Stdin, out: os.Stdout, err: os.Stderr }
	c.outTTY = term.IsTerminal(int(c.out.Fd()))
	c.errTTY = term.IsTerminal(int(c.err.Fd()))
	c.win = &winWatcher{C: make(chan struct{}, 1), ch: make(chan os.Signal, 1)}
	signal.Notify(c.win.ch, syscall.SIGWINCH)
	go func() {
		defer close(c.win.C)
		for range c.win.ch {
			select { case c.win.C <- struct{}{}: default: }
		}
	}()
	return c, nil
}

func (c *console) EnableVT() {}

func (c *console) Size() (int, int) {
	w, h, err := term.GetSize(int(c.out.Fd()))
	if err != nil { return 0, 0 }
	return w, h
}

func (c *console) MakeRaw() (RawState, error) {
	fd := int(c.in.Fd())
	st, err := term.MakeRaw(fd)
	if err != nil { return nil, err }
	r := &rawState{st: st, fd: fd}
	c.raw = r
	return r, nil
}

func (c *console) Restore(s RawState) error {
	r, ok := s.(*rawState)
	if !ok || r == nil || r.st == nil { return nil }
	err := term.Restore(r.fd, r.st)
	if err == nil { c.raw = nil }
	return err
}

func (c *console) Close() error {
	c.closeOnce.Do(func() {
		if c.win != nil && c.win.ch != nil {
			signal.Stop(c.win.ch)
			close(c.win.ch)
		}
	})
	return nil
}
